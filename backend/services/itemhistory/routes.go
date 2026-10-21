package itemhistory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const PageSize = 20

const historyQuery = `SELECT i.id,e.id,COALESCE(i.description,i.name),COALESCE(e.provider_name,''),COALESCE(e.occurred_on,(e.expense_time_utc AT TIME ZONE 'UTC')::date)::text,e.currency,(CASE WHEN i.description IS NULL THEN i.amount ELSE i.quantity END)::text,(CASE WHEN i.description IS NULL THEN i.unit_price ELSE i.confirmed_unit_price END)::text,COALESCE(i.line_total,ROUND(i.amount*i.unit_price,3))::text
        FROM expense e JOIN item i ON i.expense_id=e.id
        WHERE e.group_id=$1 AND NOT e.is_deleted AND STRPOS(LOWER(COALESCE(i.description,i.name)),LOWER($2))>0
        AND EXISTS(SELECT 1 FROM group_member m JOIN users u ON u.id=m.user_id WHERE m.group_id=e.group_id AND m.user_id=$3 AND u.is_active)
        AND (COALESCE(e.occurred_on,(e.expense_time_utc AT TIME ZONE 'UTC')::date),e.id,i.id)<($4::date,$5::uuid,$6::uuid)
        ORDER BY COALESCE(e.occurred_on,(e.expense_time_utc AT TIME ZONE 'UTC')::date) DESC,e.id DESC,i.id DESC LIMIT 21`

var ErrCursor = errors.New("invalid item history cursor")
var ErrQuery = errors.New("item description must contain 1–200 characters")

type Row struct {
	ItemID      uuid.UUID `json:"itemId"`
	ExpenseID   uuid.UUID `json:"expenseId"`
	Description string    `json:"description"`
	Merchant    string    `json:"merchant"`
	Date        string    `json:"date"`
	Currency    string    `json:"currency"`
	Quantity    *string   `json:"quantity"`
	UnitPrice   *string   `json:"unitPrice"`
	LineTotal   string    `json:"lineTotal"`
}
type Page struct {
	Items      []Row  `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
}
type cursor struct {
	Date      string    `json:"date"`
	ExpenseID uuid.UUID `json:"expense"`
	ItemID    uuid.UUID `json:"item"`
	Scope     string    `json:"scope"`
}
type Store struct{ DB *sql.DB }

func scope(group uuid.UUID, query string) string {
	hash := sha256.Sum256([]byte(group.String() + "\x00" + query))
	return hex.EncodeToString(hash[:])
}

func (s *Store) List(ctx context.Context, group, user uuid.UUID, query, after string) (Page, error) {
	page := Page{Items: []Row{}}
	var member bool
	err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM group_member m JOIN users u ON u.id=m.user_id WHERE m.group_id=$1 AND m.user_id=$2 AND u.is_active)`, group, user).Scan(&member)
	if err != nil {
		return page, err
	}
	if !member {
		return page, types.ErrGroupNotExist
	}
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) < 1 || utf8.RuneCountInString(query) > 200 {
		return page, ErrQuery
	}
	c := cursor{Date: "9999-12-31", ExpenseID: uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"), ItemID: uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"), Scope: scope(group, query)}
	if after != "" {
		if len(after) > 1024 {
			return page, ErrCursor
		}
		data, err := base64.RawURLEncoding.DecodeString(after)
		if err != nil {
			return page, ErrCursor
		}
		if json.Unmarshal(data, &c) != nil || c.Scope != scope(group, query) || c.ItemID == uuid.Nil || c.ExpenseID == uuid.Nil {
			return page, ErrCursor
		}
		if _, err := time.Parse("2006-01-02", c.Date); err != nil {
			return page, ErrCursor
		}
	}
	rows, err := s.DB.QueryContext(ctx, historyQuery, group, query, user, c.Date, c.ExpenseID, c.ItemID)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var row Row
		if err := rows.Scan(&row.ItemID, &row.ExpenseID, &row.Description, &row.Merchant, &row.Date, &row.Currency, &row.Quantity, &row.UnitPrice, &row.LineTotal); err != nil {
			return page, err
		}
		page.Items = append(page.Items, row)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	if len(page.Items) > PageSize {
		page.Items = page.Items[:PageSize]
		last := page.Items[PageSize-1]
		encoded, _ := json.Marshal(cursor{last.Date, last.ExpenseID, last.ItemID, scope(group, query)})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}

func (s *Store) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/group/:groupid/item-history", s.get)
}
func (s *Store) get(c *gin.Context) {
	id, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, 401, err)
		return
	}
	group, err := uuid.Parse(c.Param("groupid"))
	if err != nil {
		utils.WriteError(c, 404, types.ErrGroupNotExist)
		return
	}
	user, err := uuid.Parse(id)
	if err != nil {
		utils.WriteError(c, 401, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	page, err := s.List(ctx, group, user, c.Query("q"), c.Query("cursor"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, types.ErrGroupNotExist) {
			status = 404
		} else if errors.Is(err, ErrCursor) || errors.Is(err, ErrQuery) {
			status = 400
		}
		utils.WriteError(c, status, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	utils.WriteJSON(c, 200, page)
}
