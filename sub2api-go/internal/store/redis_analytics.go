package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sub2api-go/internal/model"
)

func (s *RedisStore) AggregateUserConsumeByDay(ctx context.Context, userID string, days int) ([]model.DailyUsagePoint, error) {
	if s.sqlite != nil {
		return s.sqlite.AggregateUserConsumeByDay(ctx, userID, days)
	}
	txs, err := s.scanUserTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return aggregateUserConsumeByDayFromTxs(txs, userID, days), nil
}

func (s *RedisStore) GetUsageSummary(ctx context.Context, userID string) (*model.UsageSummary, error) {
	if s.sqlite != nil {
		return s.sqlite.GetUsageSummary(ctx, userID)
	}
	txs, err := s.scanUserTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return usageSummaryFromTxs(txs, userID), nil
}

func (s *RedisStore) AggregateConsumeByModel(ctx context.Context, userID string, days int) ([]model.ModelUsageRow, error) {
	if s.sqlite != nil {
		return s.sqlite.AggregateConsumeByModel(ctx, userID, days)
	}
	txs, err := s.scanUserTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	return aggregateByModelFromTxs(txs, userID, days), nil
}

func (s *RedisStore) ListPaymentRecords(ctx context.Context, userID string, limit, offset int) ([]*model.PaymentRecord, int, error) {
	if s.sqlite != nil {
		return s.sqlite.ListPaymentRecords(ctx, userID, limit, offset)
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	txs, err := s.scanUserTransactions(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	out, total := paymentRecordsFromTxs(txs, userID, limit, offset)
	if out == nil {
		out = []*model.PaymentRecord{}
	}
	return out, total, nil
}

func (s *RedisStore) ExportUsageCSV(ctx context.Context, userID, month string) ([]byte, error) {
	if s.sqlite != nil {
		return s.sqlite.ExportUsageCSV(ctx, userID, month)
	}
	month = strings.TrimSpace(month)
	if month == "" {
		month = time.Now().UTC().Format("2006-01")
	}
	txs, err := s.scanUserTransactions(ctx, userID)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("date,key_prefix,model,input_tokens,output_tokens,amount_usd\n")
	for _, tx := range txs {
		if tx == nil || !isConsumeLedgerType(tx.Type) {
			continue
		}
		if tx.CreatedAt.UTC().Format("2006-01") != month {
			continue
		}
		prefix := ""
		if tx.KeyID != "" {
			if key, err := s.GetKeyByID(ctx, tx.KeyID); err == nil {
				prefix = key.KeyPrefix
			}
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

func (s *RedisStore) scanUserTransactions(ctx context.Context, userID string) ([]*model.Transaction, error) {
	var cursor uint64
	var all []*model.Transaction
	for {
		keys, next, err := s.client.Scan(ctx, cursor, KeyPrefixTransaction+"*", 200).Result()
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			raw, err := s.client.Get(ctx, k).Result()
			if err != nil {
				continue
			}
			var tx model.Transaction
			if json.Unmarshal([]byte(raw), &tx) != nil {
				continue
			}
			if tx.UserID == userID {
				cp := tx
				all = append(all, &cp)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return all, nil
}
