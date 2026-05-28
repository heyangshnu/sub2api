package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sub2api-go/internal/model"
)

func monthStartUTC(now time.Time) time.Time {
	y, m, _ := now.UTC().Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}

const consumeLedgerSQL = `type IN ('chat_consume', 'api_consume', 'consume')`

func (s *SQLiteStore) AggregateUserConsumeByDay(ctx context.Context, userID string, days int) ([]model.DailyUsagePoint, error) {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(created_at) AS d,
		       COALESCE(SUM(amount), 0) AS total,
		       COUNT(*) AS cnt
		FROM account_ledger
		WHERE user_id = ?
		  AND type IN ('chat_consume', 'api_consume', 'consume')
		  AND created_at >= ?
		GROUP BY date(created_at)
		ORDER BY d ASC
	`, userID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DailyUsagePoint
	for rows.Next() {
		var d string
		var total float64
		var cnt int
		if err := rows.Scan(&d, &total, &cnt); err != nil {
			return nil, err
		}
		out = append(out, model.DailyUsagePoint{
			Date:          d,
			TotalConsumed: total,
			RequestCount:  cnt,
		})
	}
	return out, rows.Err()
}

func (s *SQLiteStore) queryUsageRollup(ctx context.Context, userID string, since time.Time) (spend float64, cnt int, inTok, outTok int64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(amount), 0),
		       COUNT(*),
		       COALESCE(SUM(COALESCE(input_tokens, 0)), 0),
		       COALESCE(SUM(COALESCE(output_tokens, 0)), 0)
		FROM account_ledger
		WHERE user_id = ?
		  AND `+consumeLedgerSQL+`
		  AND created_at >= ?
	`, userID, since.Format("2006-01-02 15:04:05")).Scan(&spend, &cnt, &inTok, &outTok)
	return
}

func (s *SQLiteStore) GetUsageSummary(ctx context.Context, userID string) (*model.UsageSummary, error) {
	now := time.Now().UTC()
	todayStart := dayStartUTC(now)
	monthStart := monthStartUTC(now)
	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	tSpend, tCnt, tIn, tOut, err := s.queryUsageRollup(ctx, userID, todayStart)
	if err != nil {
		return nil, err
	}
	mSpend, mCnt, _, _, err := s.queryUsageRollup(ctx, userID, monthStart)
	if err != nil {
		return nil, err
	}
	totSpend, totCnt, totIn, totOut, err := s.queryUsageRollup(ctx, userID, epoch)
	if err != nil {
		return nil, err
	}
	_ = mCnt // month request count kept for API compat; month tile uses spend only
	return &model.UsageSummary{
		TodaySpendUSD:     tSpend,
		TodayRequestCount: tCnt,
		TodayInputTokens:  tIn,
		TodayOutputTokens: tOut,
		MonthSpendUSD:     mSpend,
		MonthRequestCount: mCnt,
		TotalSpendUSD:     totSpend,
		TotalRequestCount: totCnt,
		TotalInputTokens:  totIn,
		TotalOutputTokens: totOut,
	}, nil
}

func (s *SQLiteStore) AggregateConsumeByModel(ctx context.Context, userID string, days int) ([]model.ModelUsageRow, error) {
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(model, ''), 'unknown') AS model,
		       COUNT(*) AS cnt,
		       COALESCE(SUM(COALESCE(input_tokens, 0)), 0),
		       COALESCE(SUM(COALESCE(output_tokens, 0)), 0),
		       COALESCE(SUM(amount), 0)
		FROM account_ledger
		WHERE user_id = ?
		  AND type IN ('chat_consume', 'api_consume', 'consume')
		  AND created_at >= ?
		GROUP BY COALESCE(NULLIF(model, ''), 'unknown')
		ORDER BY SUM(amount) DESC
	`, userID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ModelUsageRow
	for rows.Next() {
		var row model.ModelUsageRow
		if err := rows.Scan(&row.Model, &row.RequestCount, &row.InputTokens, &row.OutputTokens, &row.TotalConsumed); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListPaymentRecords(ctx context.Context, userID string, limit, offset int) ([]*model.PaymentRecord, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM account_ledger
		WHERE user_id = ? AND type IN ('topup', 'admin_topup')
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, amount, COALESCE(stripe_payment_id, ''), COALESCE(note, ''), created_at
		FROM account_ledger
		WHERE user_id = ? AND type IN ('topup', 'admin_topup')
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*model.PaymentRecord
	for rows.Next() {
		var p model.PaymentRecord
		var stripeID string
		if err := rows.Scan(&p.ID, &p.Amount, &stripeID, &p.Note, &p.CreatedAt); err != nil {
			return nil, 0, err
		}
		p.Status = "completed"
		p.StripeSessionID = stripeID
		out = append(out, &p)
	}
	if out == nil {
		out = []*model.PaymentRecord{}
	}
	return out, total, rows.Err()
}

func (s *SQLiteStore) ExportUsageCSV(ctx context.Context, userID, month string) ([]byte, error) {
	month = strings.TrimSpace(month)
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.created_at,
		       COALESCE(k.key_prefix, ''),
		       COALESCE(l.model, ''),
		       COALESCE(l.input_tokens, 0),
		       COALESCE(l.output_tokens, 0),
		       l.amount
		FROM account_ledger l
		LEFT JOIN api_keys k ON l.key_id = k.id
		WHERE l.user_id = ?
		  AND l.type IN ('chat_consume', 'api_consume', 'consume')
		  AND strftime('%Y-%m', l.created_at) = ?
		ORDER BY l.created_at ASC
	`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString("date,key_prefix,model,input_tokens,output_tokens,amount_usd\n")
	for rows.Next() {
		var created time.Time
		var prefix, modelName string
		var inTok, outTok int
		var amount float64
		if err := rows.Scan(&created, &prefix, &modelName, &inTok, &outTok, &amount); err != nil {
			return nil, err
		}
		b.WriteString(fmt.Sprintf("%s,%s,%s,%d,%d,%.6f\n",
			created.UTC().Format(time.RFC3339),
			escapeCSV(prefix),
			escapeCSV(modelName),
			inTok,
			outTok,
			amount,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// AggregateConsumeByDay aggregates per-key consume from the ledger.
func (s *SQLiteStore) AggregateConsumeByDay(ctx context.Context, keyHash string, days int) ([]model.DailyUsagePoint, error) {
	key, err := s.GetKeyByHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}
	return s.aggregateConsumeByDayForKeyID(ctx, key.ID, days)
}

func (s *SQLiteStore) aggregateConsumeByDayForKeyID(ctx context.Context, keyID string, days int) ([]model.DailyUsagePoint, error) {
	if days < 1 {
		days = 14
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	rows, err := s.db.QueryContext(ctx, `
		SELECT date(created_at) AS d,
		       COALESCE(SUM(amount), 0),
		       COUNT(*)
		FROM account_ledger
		WHERE key_id = ?
		  AND type IN ('chat_consume', 'api_consume', 'consume')
		  AND created_at >= ?
		GROUP BY date(created_at)
		ORDER BY d ASC
	`, keyID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.DailyUsagePoint
	for rows.Next() {
		var d string
		var total float64
		var cnt int
		if err := rows.Scan(&d, &total, &cnt); err != nil {
			return nil, err
		}
		out = append(out, model.DailyUsagePoint{Date: d, TotalConsumed: total, RequestCount: cnt})
	}
	return out, rows.Err()
}
