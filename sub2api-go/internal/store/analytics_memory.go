package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sub2api-go/internal/model"
)

func (s *MemoryStore) memoryUserTxs(userID string) []*model.Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*model.Transaction
	for i := range s.transactions {
		if s.transactions[i].UserID == userID {
			cp := s.transactions[i]
			out = append(out, &cp)
		}
	}
	return out
}

func (s *MemoryStore) AggregateUserConsumeByDay(ctx context.Context, userID string, days int) ([]model.DailyUsagePoint, error) {
	_ = ctx
	return aggregateUserConsumeByDayFromTxs(s.memoryUserTxs(userID), userID, days), nil
}

func (s *MemoryStore) GetUsageSummary(ctx context.Context, userID string) (*model.UsageSummary, error) {
	_ = ctx
	return usageSummaryFromTxs(s.memoryUserTxs(userID), userID), nil
}

func (s *MemoryStore) AggregateConsumeByModel(ctx context.Context, userID string, days int) ([]model.ModelUsageRow, error) {
	_ = ctx
	return aggregateByModelFromTxs(s.memoryUserTxs(userID), userID, days), nil
}

func (s *MemoryStore) ListPaymentRecords(ctx context.Context, userID string, limit, offset int) ([]*model.PaymentRecord, int, error) {
	_ = ctx
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	out, total := paymentRecordsFromTxs(s.memoryUserTxs(userID), userID, limit, offset)
	if out == nil {
		out = []*model.PaymentRecord{}
	}
	return out, total, nil
}

func (s *MemoryStore) ExportUsageCSV(ctx context.Context, userID, month string) ([]byte, error) {
	_ = ctx
	month = strings.TrimSpace(month)
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	var b strings.Builder
	b.WriteString("date,key_prefix,model,input_tokens,output_tokens,amount_usd\n")
	for _, tx := range s.memoryUserTxs(userID) {
		if tx == nil || !isConsumeLedgerType(tx.Type) {
			continue
		}
		if tx.CreatedAt.UTC().Format("2006-01") != month {
			continue
		}
		prefix := ""
		if tx.KeyID != "" {
			s.mu.RLock()
			for _, k := range s.keys {
				if k.ID == tx.KeyID {
					prefix = k.KeyPrefix
					break
				}
			}
			s.mu.RUnlock()
		}
		b.WriteString(fmt.Sprintf("%s,%s,%s,%d,%d,%.6f\n",
			tx.CreatedAt.UTC().Format(time.RFC3339),
			escapeCSV(prefix),
			escapeCSV(tx.Model),
			tx.InputTokens,
			tx.OutputTokens,
			tx.Amount,
		))
	}
	return []byte(b.String()), nil
}
