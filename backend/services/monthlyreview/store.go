package monthlyreview

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"expense-tracker/backend/types"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxPendingWebPushDeliveries  = 1000
	monthlyReviewExpensePageSize = 20
)

var ErrInvalidExpenseCursor = errors.New("invalid monthly review expense cursor")

var earliestMonthlyReviewMonth = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) PublishEligible(ctx context.Context, now time.Time, limit int) (PublicationResult, error) {
	if limit < 1 || limit > 100 {
		return PublicationResult{}, errors.New("monthly review publication limit must be between 1 and 100")
	}
	monthStart := firstOfMonth(now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PublicationResult{}, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT expense.group_id,
			date_trunc('month', COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date))::date AS review_month
		FROM expense
		JOIN groups ON groups.id = expense.group_id
		LEFT JOIN monthly_review_publication publication
			ON publication.group_id = expense.group_id
			AND publication.review_month = date_trunc('month', COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date))::date
		WHERE groups.is_active IS TRUE
			AND groups.group_type = 'home'
			AND expense.is_deleted IS FALSE
			AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < $1
			AND publication.group_id IS NULL
		GROUP BY expense.group_id,
			date_trunc('month', COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date))::date
		ORDER BY review_month, expense.group_id
		LIMIT $2`, monthStart, limit)
	if err != nil {
		return PublicationResult{}, err
	}
	type candidate struct {
		groupID uuid.UUID
		month   time.Time
	}
	candidates := make([]candidate, 0, limit)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.groupID, &item.month); err != nil {
			rows.Close()
			return PublicationResult{}, err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Close(); err != nil {
		return PublicationResult{}, err
	}
	if err := rows.Err(); err != nil {
		return PublicationResult{}, err
	}

	result := PublicationResult{}
	for _, candidate := range candidates {
		var inserted bool
		if err := tx.QueryRowContext(ctx, `
			WITH publication AS (
				INSERT INTO monthly_review_publication (group_id, review_month)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
				RETURNING TRUE
			)
			SELECT COALESCE(BOOL_OR(TRUE), FALSE) FROM publication`, candidate.groupID, candidate.month).Scan(&inserted); err != nil {
			return PublicationResult{}, err
		}
		if !inserted {
			continue
		}
		result.Published++
		queued, eligible, err := queueMonthlyReviewNotifications(ctx, tx, candidate.groupID, candidate.month)
		if err != nil {
			return PublicationResult{}, err
		}
		if queued != eligible {
			return PublicationResult{}, errors.New("monthly review notification capacity exhausted")
		}
		result.Deliveries += queued
	}

	if err := tx.Commit(); err != nil {
		return PublicationResult{}, err
	}
	return result, nil
}

func queueMonthlyReviewNotifications(ctx context.Context, tx *sql.Tx, groupID uuid.UUID, month time.Time) (int, int, error) {
	var queued, eligible int
	err := tx.QueryRowContext(ctx, `
		WITH capacity AS (
			SELECT GREATEST($3 - COUNT(*), 0)::bigint AS remaining
			FROM web_push_delivery
			WHERE completed_at IS NULL AND expires_at > NOW()
		), recipients AS (
			SELECT subscription.id, subscription.user_id, groups.group_name
			FROM web_push_subscription subscription
			JOIN group_member member ON member.user_id = subscription.user_id
			JOIN groups ON groups.id = member.group_id AND groups.is_active IS TRUE
			LEFT JOIN group_notification_preference preference
				ON preference.group_id = member.group_id AND preference.user_id = member.user_id
			WHERE member.group_id = $1
				AND COALESCE(preference.muted, FALSE) IS FALSE
		), inserted AS (
			INSERT INTO web_push_delivery (
			notification_type, review_month, subscription_id, recipient_user_id,
			group_id, group_name, expires_at
			)
			SELECT 'monthly_review_available', $2, id, user_id, $1, group_name,
				NOW() + INTERVAL '7 days'
			FROM recipients
			WHERE (SELECT COUNT(*) FROM recipients) <= (SELECT remaining FROM capacity)
			ON CONFLICT DO NOTHING
			RETURNING 1
		)
		SELECT (SELECT COUNT(*) FROM inserted), (SELECT COUNT(*) FROM recipients)`,
		groupID, month, maxPendingWebPushDeliveries).Scan(&queued, &eligible)
	if err != nil {
		return 0, 0, err
	}
	return queued, eligible, nil
}

func (s *Store) Get(ctx context.Context, groupID, userID uuid.UUID, month time.Time, now time.Time) (Review, error) {
	review := Review{
		GroupID:    groupID.String(),
		Month:      month.Format("2006-01"),
		Currencies: make([]CurrencySummary, 0),
	}
	groupName, err := s.authorizedGroupName(ctx, groupID, userID)
	if err != nil {
		return Review{}, err
	}
	review.GroupName = groupName
	if !month.Before(firstOfMonth(now)) {
		review.State = StateNotClosed
		return review, nil
	}

	var publishedAt sql.NullTime
	if err := s.db.QueryRowContext(ctx, `
		SELECT published_at FROM monthly_review_publication
		WHERE group_id = $1 AND review_month = $2`, groupID, month).Scan(&publishedAt); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Review{}, err
	}

	summaries, err := s.currencySummaries(ctx, groupID, month)
	if err != nil {
		return Review{}, err
	}
	if len(summaries) == 0 {
		review.State = StateEmpty
		return review, nil
	}
	if !publishedAt.Valid {
		review.State = StateUnpublished
		return review, nil
	}

	nets, err := s.memberNetTotals(ctx, groupID, month)
	if err != nil {
		return Review{}, err
	}

	review.State = StatePublished
	review.PublishedAt = publishedAt.Time.UTC()
	for i := range summaries {
		summaries[i].MemberNetTotals = nets[summaries[i].Currency]
		if summaries[i].MemberNetTotals == nil {
			summaries[i].MemberNetTotals = make([]UserAmount, 0)
		}
	}
	review.Currencies = summaries
	return review, nil
}

func (s *Store) authorizedGroupName(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	var groupName string
	if err := s.db.QueryRowContext(ctx, `
		SELECT groups.group_name
		FROM groups
		JOIN group_member ON group_member.group_id = groups.id AND group_member.user_id = $2
		WHERE groups.id = $1 AND groups.is_active IS TRUE AND groups.group_type = 'home'`, groupID, userID).
		Scan(&groupName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrGroupNotExist
		}
		return "", err
	}
	return groupName, nil
}

func (s *Store) GetTrend(ctx context.Context, groupID, userID uuid.UUID, endMonth, now time.Time) (Trend, error) {
	groupName, err := s.authorizedGroupName(ctx, groupID, userID)
	if err != nil {
		return Trend{}, err
	}
	latestClosedMonth := firstOfMonth(now).AddDate(0, -1, 0)
	if endMonth.After(latestClosedMonth) {
		endMonth = latestClosedMonth
	}
	if endMonth.Before(earliestMonthlyReviewMonth) {
		endMonth = earliestMonthlyReviewMonth
	}
	startMonth := endMonth.AddDate(0, -11, 0)
	if startMonth.Before(earliestMonthlyReviewMonth) {
		startMonth = earliestMonthlyReviewMonth
	}
	trend := Trend{
		GroupID: groupID.String(), GroupName: groupName,
		StartMonth: startMonth.Format("2006-01"), EndMonth: endMonth.Format("2006-01"),
		Currencies: make([]TrendCurrency, 0),
	}

	var latestReport sql.NullTime
	if err := s.db.QueryRowContext(ctx, `
		SELECT MAX(publication.review_month)
		FROM monthly_review_publication publication
		WHERE publication.group_id = $1
			AND publication.review_month <= $2
			AND publication.review_month >= $3
			AND EXISTS (
				SELECT 1 FROM expense
				WHERE expense.group_id = publication.group_id
					AND expense.is_deleted IS FALSE
					AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= publication.review_month
					AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < publication.review_month + INTERVAL '1 month'
			)`, groupID, latestClosedMonth, earliestMonthlyReviewMonth).Scan(&latestReport); err != nil {
		return Trend{}, err
	}
	if latestReport.Valid {
		trend.LatestReportMonth = latestReport.Time.Format("2006-01")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT date_trunc('month', COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date))::date,
			expense.currency, SUM(expense.total)::text, COUNT(*)
		FROM expense
		JOIN monthly_review_publication publication
			ON publication.group_id = expense.group_id
			AND publication.review_month = date_trunc('month', COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date))::date
		WHERE expense.group_id = $1 AND expense.is_deleted IS FALSE
			AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= $2
			AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < ($3::date + INTERVAL '1 month')
		GROUP BY 1, expense.currency
		ORDER BY expense.currency, 1`, groupID, startMonth, endMonth)
	if err != nil {
		return Trend{}, err
	}
	defer rows.Close()

	byCurrency := make(map[string]map[string]TrendMonth)
	for rows.Next() {
		var month time.Time
		var currency string
		var item TrendMonth
		if err := rows.Scan(&month, &currency, &item.Total, &item.ExpenseCount); err != nil {
			return Trend{}, err
		}
		total, err := decimal.NewFromString(item.Total)
		if err != nil {
			return Trend{}, fmt.Errorf("parse monthly trend total: %w", err)
		}
		item.Total = total.String()
		item.Month = month.Format("2006-01")
		if byCurrency[currency] == nil {
			byCurrency[currency] = make(map[string]TrendMonth)
		}
		byCurrency[currency][item.Month] = item
	}
	if err := rows.Err(); err != nil {
		return Trend{}, err
	}

	currencies := make([]string, 0, len(byCurrency))
	for currency := range byCurrency {
		currencies = append(currencies, currency)
	}
	slices.Sort(currencies)
	for _, currency := range currencies {
		months := make([]TrendMonth, 0, 12)
		for offset := 0; ; offset++ {
			month := startMonth.AddDate(0, offset, 0).Format("2006-01")
			if month > trend.EndMonth {
				break
			}
			item, ok := byCurrency[currency][month]
			if !ok {
				item = TrendMonth{Month: month, Total: "0"}
			}
			months = append(months, item)
		}
		trend.Currencies = append(trend.Currencies, TrendCurrency{Currency: currency, Months: months})
	}
	return trend, nil
}

func (s *Store) currencySummaries(ctx context.Context, groupID uuid.UUID, month time.Time) ([]CurrencySummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT expense.currency,
			CASE
				WHEN GROUPING(expense_type.category) = 0 THEN 'category'
				WHEN GROUPING(expense.pay_by_user_id) = 0 THEN 'payer'
				ELSE 'total'
			END,
			COALESCE(expense_type.category, ''),
			COALESCE(expense.pay_by_user_id::text, ''),
			COALESCE(users.username, ''),
			SUM(expense.total)::text,
			CASE
				WHEN GROUPING(expense_type.category) = 1 AND GROUPING(expense.pay_by_user_id) = 1
				THEN COUNT(*)::int
				ELSE 0
			END
		FROM expense
		JOIN expense_type ON expense_type.id = expense.exp_type_id
		JOIN users ON users.id = expense.pay_by_user_id
		WHERE expense.group_id = $1 AND expense.is_deleted IS FALSE
			AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= $2
			AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < ($2::date + INTERVAL '1 month')
		GROUP BY GROUPING SETS (
			(expense.currency),
			(expense.currency, expense_type.category),
			(expense.currency, expense.pay_by_user_id, users.username)
		)
		ORDER BY 1, 2, 3, 5, 4`, groupID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byCurrency := make(map[string]*CurrencySummary)
	for rows.Next() {
		var currency, kind, name, userID, username, amount string
		var expenseCount int
		if err := rows.Scan(&currency, &kind, &name, &userID, &username, &amount, &expenseCount); err != nil {
			return nil, err
		}
		parsedAmount, err := decimal.NewFromString(amount)
		if err != nil {
			return nil, fmt.Errorf("parse monthly review amount: %w", err)
		}
		summary := byCurrency[currency]
		if summary == nil {
			summary = &CurrencySummary{
				Currency: currency, Categories: make([]NamedAmount, 0),
				Payers: make([]UserAmount, 0), MemberNetTotals: make([]UserAmount, 0),
			}
			byCurrency[currency] = summary
		}
		switch kind {
		case "total":
			summary.Total = parsedAmount.String()
			summary.ExpenseCount = expenseCount
		case "category":
			summary.Categories = append(summary.Categories, NamedAmount{Name: name, Amount: parsedAmount.String()})
		case "payer":
			summary.Payers = append(summary.Payers, UserAmount{UserID: userID, Username: username, Amount: parsedAmount.String()})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	currencies := slices.Sorted(maps.Keys(byCurrency))
	summaries := make([]CurrencySummary, 0, len(currencies))
	for _, currency := range currencies {
		summaries = append(summaries, *byCurrency[currency])
	}
	return summaries, nil
}

type expenseCursor struct {
	OccurredOn     time.Time `json:"occurredOn"`
	ExpenseTimeUTC time.Time `json:"expenseTimeUtc"`
	ID             uuid.UUID `json:"id"`
}

func decodeExpenseCursor(value string) (expenseCursor, error) {
	if value == "" {
		return expenseCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return expenseCursor{}, ErrInvalidExpenseCursor
	}
	var cursor expenseCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.OccurredOn.IsZero() || cursor.ExpenseTimeUTC.IsZero() || cursor.ID == uuid.Nil {
		return expenseCursor{}, ErrInvalidExpenseCursor
	}
	return cursor, nil
}

func encodeExpenseCursor(cursor expenseCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func (s *Store) ListExpenses(ctx context.Context, groupID, userID uuid.UUID, month time.Time, currency, cursorValue string) (ExpensePage, error) {
	if _, err := s.authorizedGroupName(ctx, groupID, userID); err != nil {
		return ExpensePage{}, err
	}
	cursor, err := decodeExpenseCursor(cursorValue)
	if err != nil {
		return ExpensePage{}, err
	}

	var cursorOccurredOn, cursorExpenseTime, cursorID any
	if cursorValue != "" {
		cursorOccurredOn, cursorExpenseTime, cursorID = cursor.OccurredOn, cursor.ExpenseTimeUTC, cursor.ID
	}
	rows, err := s.db.QueryContext(ctx, `
        SELECT expense.id, expense.description,
            COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date),
            expense.expense_time_utc, expense_type.name, expense_type.category,
            expense.pay_by_user_id, users.username, expense.total::text, expense.is_settled
        FROM expense
        JOIN expense_type ON expense_type.id = expense.exp_type_id
        JOIN users ON users.id = expense.pay_by_user_id
        JOIN monthly_review_publication publication
            ON publication.group_id = expense.group_id AND publication.review_month = $2
        WHERE expense.group_id = $1 AND expense.is_deleted IS FALSE AND expense.currency = $3
            AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= $2
            AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < ($2::date + INTERVAL '1 month')
            AND ($4::date IS NULL OR (
                COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date),
                expense.expense_time_utc, expense.id
            ) < ($4::date, $5::timestamptz, $6::uuid))
        ORDER BY COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) DESC,
            expense.expense_time_utc DESC, expense.id DESC
        LIMIT $7`, groupID, month, currency, cursorOccurredOn, cursorExpenseTime, cursorID, monthlyReviewExpensePageSize+1)
	if err != nil {
		return ExpensePage{}, err
	}
	defer rows.Close()

	type pageRow struct {
		expense Expense
		cursor  expenseCursor
	}
	pageRows := make([]pageRow, 0, monthlyReviewExpensePageSize+1)
	for rows.Next() {
		var row pageRow
		var expenseID, payerID uuid.UUID
		if err := rows.Scan(&expenseID, &row.expense.Description, &row.cursor.OccurredOn,
			&row.cursor.ExpenseTimeUTC, &row.expense.ExpenseType, &row.expense.Category,
			&payerID, &row.expense.PayerName, &row.expense.Total, &row.expense.Settled); err != nil {
			return ExpensePage{}, err
		}
		row.cursor.ID = expenseID
		row.expense.ID = expenseID.String()
		row.expense.PayerID = payerID.String()
		row.expense.OccurredOn = row.cursor.OccurredOn.Format(time.DateOnly)
		pageRows = append(pageRows, row)
	}
	if err := rows.Err(); err != nil {
		return ExpensePage{}, err
	}

	page := ExpensePage{Expenses: make([]Expense, 0, min(len(pageRows), monthlyReviewExpensePageSize))}
	for _, row := range pageRows[:min(len(pageRows), monthlyReviewExpensePageSize)] {
		page.Expenses = append(page.Expenses, row.expense)
	}
	if len(pageRows) > monthlyReviewExpensePageSize {
		nextCursor, err := encodeExpenseCursor(pageRows[monthlyReviewExpensePageSize-1].cursor)
		if err != nil {
			return ExpensePage{}, err
		}
		page.NextCursor = nextCursor
	}
	return page, nil
}

func (s *Store) memberNetTotals(ctx context.Context, groupID uuid.UUID, month time.Time) (map[string][]UserAmount, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH entries AS (
			SELECT expense.currency, ledger.lender_user_id AS user_id, ledger.share AS amount
			FROM ledger JOIN expense ON expense.id = ledger.expense_id
			WHERE expense.group_id = $1 AND expense.is_deleted IS FALSE
				AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= $2
				AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < ($2::date + INTERVAL '1 month')
			UNION ALL
			SELECT expense.currency, ledger.borrower_user_id, -ledger.share
			FROM ledger JOIN expense ON expense.id = ledger.expense_id
			WHERE expense.group_id = $1 AND expense.is_deleted IS FALSE
				AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) >= $2
				AND COALESCE(expense.occurred_on, (expense.expense_time_utc AT TIME ZONE 'UTC')::date) < ($2::date + INTERVAL '1 month')
		)
		SELECT entries.currency, entries.user_id, users.username, SUM(entries.amount)::text
		FROM entries JOIN users ON users.id = entries.user_id
		GROUP BY entries.currency, entries.user_id, users.username
		ORDER BY entries.currency, users.username, entries.user_id`, groupID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	totals := make(map[string][]UserAmount)
	for rows.Next() {
		var currency string
		var id uuid.UUID
		var total UserAmount
		if err := rows.Scan(&currency, &id, &total.Username, &total.Amount); err != nil {
			return nil, err
		}
		total.UserID = id.String()
		totals[currency] = append(totals[currency], total)
	}
	return totals, rows.Err()
}

func firstOfMonth(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func ParseMonth(value string) (time.Time, error) {
	month, err := time.Parse("2006-01", value)
	if err != nil || month.Format("2006-01") != value {
		return time.Time{}, fmt.Errorf("invalid review month")
	}
	return month.UTC(), nil
}
