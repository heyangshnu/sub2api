package store

import (
	"sort"
	"time"

	"sub2api-go/internal/model"
)

func aggregateUserConsumeByDayFromTxs(txs []*model.Transaction, userID string, days int) []model.DailyUsagePoint {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	byDay := make(map[string]*model.DailyUsagePoint)
	for _, tx := range txs {
		if tx == nil || tx.UserID != userID || !isConsumeLedgerType(tx.Type) {
			continue
		}
		if tx.CreatedAt.UTC().Before(cutoff) {
			continue
		}
		d := tx.CreatedAt.UTC().Format("2006-01-02")
		if byDay[d] == nil {
			byDay[d] = &model.DailyUsagePoint{Date: d}
		}
		byDay[d].TotalConsumed += tx.Amount
		byDay[d].RequestCount++
	}
	return sortedDailyPoints(byDay)
}

func aggregateKeyConsumeByDayFromTxs(txs []*model.Transaction, keyID string, days int) []model.DailyUsagePoint {
	if days < 1 {
		days = 14
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Truncate(24 * time.Hour)
	byDay := make(map[string]*model.DailyUsagePoint)
	for _, tx := range txs {
		if tx == nil || tx.KeyID != keyID || !isConsumeLedgerType(tx.Type) {
			continue
		}
		if tx.CreatedAt.UTC().Before(cutoff) {
			continue
		}
		d := tx.CreatedAt.UTC().Format("2006-01-02")
		if byDay[d] == nil {
			byDay[d] = &model.DailyUsagePoint{Date: d}
		}
		byDay[d].TotalConsumed += tx.Amount
		byDay[d].RequestCount++
	}
	return sortedDailyPoints(byDay)
}

func dayStartUTC(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func usageSummaryFromTxs(txs []*model.Transaction, userID string) *model.UsageSummary {
	now := time.Now().UTC()
	todayStart := dayStartUTC(now)
	monthStart := monthStartUTC(now)
	var out model.UsageSummary
	for _, tx := range txs {
		if tx == nil || tx.UserID != userID || !isConsumeLedgerType(tx.Type) {
			continue
		}
		at := tx.CreatedAt.UTC()
		in := int64(tx.InputTokens)
		outTok := int64(tx.OutputTokens)
		out.TotalSpendUSD += tx.Amount
		out.TotalRequestCount++
		out.TotalInputTokens += in
		out.TotalOutputTokens += outTok
		if !at.Before(monthStart) {
			out.MonthSpendUSD += tx.Amount
			out.MonthRequestCount++
		}
		if !at.Before(todayStart) {
			out.TodaySpendUSD += tx.Amount
			out.TodayRequestCount++
			out.TodayInputTokens += in
			out.TodayOutputTokens += outTok
		}
	}
	return &out
}

func aggregateByModelFromTxs(txs []*model.Transaction, userID string, days int) []model.ModelUsageRow {
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	type agg struct {
		row model.ModelUsageRow
	}
	byModel := make(map[string]*agg)
	for _, tx := range txs {
		if tx == nil || tx.UserID != userID || !isConsumeLedgerType(tx.Type) {
			continue
		}
		if tx.CreatedAt.UTC().Before(cutoff) {
			continue
		}
		m := tx.Model
		if m == "" {
			m = "unknown"
		}
		if byModel[m] == nil {
			byModel[m] = &agg{row: model.ModelUsageRow{Model: m}}
		}
		byModel[m].row.RequestCount++
		byModel[m].row.InputTokens += int64(tx.InputTokens)
		byModel[m].row.OutputTokens += int64(tx.OutputTokens)
		byModel[m].row.TotalConsumed += tx.Amount
	}
	out := make([]model.ModelUsageRow, 0, len(byModel))
	for _, a := range byModel {
		out = append(out, a.row)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TotalConsumed > out[j].TotalConsumed
	})
	return out
}

func paymentRecordsFromTxs(txs []*model.Transaction, userID string, limit, offset int) ([]*model.PaymentRecord, int) {
	var topups []*model.Transaction
	for _, tx := range txs {
		if tx == nil || tx.UserID != userID || !isTopupLedgerType(tx.Type) {
			continue
		}
		cp := *tx
		topups = append(topups, &cp)
	}
	sort.Slice(topups, func(i, j int) bool {
		return topups[i].CreatedAt.After(topups[j].CreatedAt)
	})
	total := len(topups)
	if offset >= total {
		return []*model.PaymentRecord{}, total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	out := make([]*model.PaymentRecord, 0, end-offset)
	for _, tx := range topups[offset:end] {
		out = append(out, &model.PaymentRecord{
			ID:              tx.ID,
			Amount:          tx.Amount,
			Status:          "completed",
			StripeSessionID: tx.StripePaymentID,
			Note:            tx.Note,
			CreatedAt:       tx.CreatedAt,
		})
	}
	return out, total
}

func sortedDailyPoints(byDay map[string]*model.DailyUsagePoint) []model.DailyUsagePoint {
	dates := make([]string, 0, len(byDay))
	for d := range byDay {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]model.DailyUsagePoint, 0, len(dates))
	for _, d := range dates {
		out = append(out, *byDay[d])
	}
	return out
}
