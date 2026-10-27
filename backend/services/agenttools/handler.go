// Package agenttools owns the Worker's read-only agent boundary. IAM authority
// comes only from Lambda's trusted API Gateway context, never HTTP headers.
package agenttools

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/internal/agentcontract"
	"github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/gin-gonic/gin"
	"uuid"
)

type Users interface {
	GetUserByID(string) (*types.User, error)
}
type Groups interface {
	CheckGroupUserPairExist(string, string) (bool, error)
}
type Balances interface {
	GetBalanceByGroupId(string) ([]types.Balance, error)
}

type Handler struct {
	users    Users
	groups   Groups
	balances Balances
	roleARN  string
	local    bool
}

// An empty or malformed role disables access. Only explicitly registered GET
// route templates are accepted, including when invoked via a Gateway fallback.
func NewHandler(users Users, groups Groups, balances Balances, roleARN string) *Handler {
	return &Handler{users: users, groups: groups, balances: balances, roleARN: roleARN}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET(agentcontract.PreflightPath, h.authorize, h.preflight)
	router.GET(agentcontract.Prefix+"/balance/:groupId", h.authorize, h.balance)
}

var rolePattern = regexp.MustCompile(`^arn:(aws|aws-us-gov|aws-cn):iam::([0-9]{12}):role/([A-Za-z0-9_+=,.@/-]+)$`)

func (h *Handler) trustedPrincipal(c *gin.Context) bool {
	role := rolePattern.FindStringSubmatch(h.roleARN)
	gateway, ok := core.GetAPIGatewayV2ContextFromContext(c.Request.Context())
	if len(role) != 4 || !ok || gateway.Authorizer == nil || gateway.Authorizer.IAM == nil {
		return false
	}
	route := "GET " + strings.ReplaceAll(c.FullPath(), ":groupId", "{groupId}")
	if c.Request.Method != http.MethodGet || gateway.HTTP.Method != http.MethodGet || gateway.RouteKey != route {
		return false
	}
	iam := gateway.Authorizer.IAM
	// STS assumed-role ARNs use the role name without its IAM path.
	roleName := role[3]
	if _, name, found := strings.CutLast(roleName, "/"); found {
		roleName = name
	}
	prefix := "arn:" + role[1] + ":sts::" + role[2] + ":assumed-role/" + roleName + "/"
	session, found := strings.CutPrefix(iam.UserARN, prefix)
	return iam.AccountID == role[2] && found && session != "" && !strings.Contains(session, "/")
}

func deny(c *gin.Context, status int) {
	c.AbortWithStatusJSON(status, gin.H{"error": http.StatusText(status)})
}

func (h *Handler) authorize(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	principalAllowed := h.trustedPrincipal(c)
	if h.local {
		principalAllowed = localBoundary(c)
	}
	if !principalAllowed {
		deny(c, http.StatusForbidden)
		return
	}
	values := c.Request.Header.Values(agentcontract.AccessTokenHeader)
	if len(values) != 1 || len(values[0]) == 0 || len(values[0]) > 8192 {
		deny(c, http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseTokenString(values[0], "access")
	if err != nil || claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		deny(c, http.StatusUnauthorized)
		return
	}
	if _, err := uuid.Parse(claims.UserID); err != nil {
		deny(c, http.StatusUnauthorized)
		return
	}
	user, err := h.users.GetUserByID(claims.UserID)
	if err != nil || user == nil || !user.IsActive {
		deny(c, http.StatusForbidden)
		return
	}
	// No cookie/bearer fallback. SigV4 owns Authorization; only this dedicated
	// header establishes the actor, independently on each request.
	c.Set("agentActor", agentcontract.Preflight{ActorID: claims.UserID, ExpiresAt: claims.ExpiresAt.Time.UTC()})
	c.Next()
}

func (h *Handler) preflight(c *gin.Context) {
	if c.Request.URL.RawQuery != "" || c.Request.ContentLength != 0 {
		deny(c, http.StatusBadRequest)
		return
	}
	actor, _ := c.Get("agentActor")
	c.JSON(http.StatusOK, actor)
}

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func (h *Handler) balance(c *gin.Context) {
	groupID := c.Param("groupId")
	if _, err := uuid.Parse(groupID); err != nil || c.Request.URL.RawQuery != "" || c.Request.ContentLength != 0 {
		deny(c, http.StatusBadRequest)
		return
	}
	actorValue, _ := c.Get("agentActor")
	actor := actorValue.(agentcontract.Preflight)
	member, err := h.groups.CheckGroupUserPairExist(groupID, actor.ActorID)
	if err != nil {
		deny(c, http.StatusServiceUnavailable)
		return
	}
	if !member {
		deny(c, http.StatusNotFound)
		return
	}
	rows, err := h.balances.GetBalanceByGroupId(groupID)
	if err != nil {
		deny(c, http.StatusServiceUnavailable)
		return
	}
	if len(rows) > agentcontract.MaxBalanceRows {
		deny(c, http.StatusUnprocessableEntity)
		return
	}
	result := agentcontract.BalanceResult{GroupID: groupID, Balances: make([]agentcontract.Balance, 0, len(rows))}
	for _, row := range rows {
		amount := row.Share.String()
		if !amountPattern.MatchString(amount) || !currencyPattern.MatchString(row.Currency) || row.GroupID.String() != groupID {
			deny(c, http.StatusInternalServerError)
			return
		}
		// Participant IDs are sufficient to identify endpoints without sending
		// account profile fields or unrestricted financial histories to models.
		from, to := row.SenderUserID.String(), row.ReceiverUserID.String()
		result.Balances = append(result.Balances, agentcontract.Balance{From: from, To: to, Amount: amount, Currency: row.Currency})
	}
	c.JSON(http.StatusOK, result)
}
