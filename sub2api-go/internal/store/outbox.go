package store

import (
	"context"
	"encoding/json"
	"log"

	"sub2api-go/internal/model"
)

// ProcessOutbox retries failed SQLite write-through rows.
func (s *Syncer) ProcessOutbox(ctx context.Context) (int, error) {
	if s.sqlite == nil {
		return 0, nil
	}
	rows, err := s.sqlite.ListPendingOutbox(ctx, 100)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, row := range rows {
		var perr error
		switch row.EntityType {
		case "ledger":
			var tx model.Transaction
			if uerr := json.Unmarshal([]byte(row.PayloadJSON), &tx); uerr != nil {
				perr = uerr
			} else {
				perr = s.sqlite.SaveLedgerEntry(ctx, &tx)
			}
		case "request_log":
			var entry model.RequestLogEntry
			if json.Unmarshal([]byte(row.PayloadJSON), &entry) == nil {
				perr = s.sqlite.SaveRequestLog(ctx, &entry)
			}
		case "user_account":
			var payload struct {
				UserID    string  `json:"user_id"`
				Spendable float64 `json:"spendable"`
				Recharged float64 `json:"recharged"`
			}
			if json.Unmarshal([]byte(row.PayloadJSON), &payload) == nil {
				user, err := s.redis.GetUserByID(ctx, payload.UserID)
				if err == nil {
					perr = s.sqlite.SaveUserAccount(ctx, user, payload.Spendable, payload.Recharged)
				} else {
					perr = err
				}
			}
		case "api_key":
			var key model.APIKey
			if json.Unmarshal([]byte(row.PayloadJSON), &key) == nil {
				perr = s.sqlite.SaveKey(ctx, &key)
			}
		default:
			log.Printf("[Outbox] unknown entity_type %s", row.EntityType)
			_ = s.sqlite.MarkOutboxProcessed(ctx, row.ID)
			processed++
			continue
		}
		if perr != nil {
			_ = s.sqlite.MarkOutboxFailed(ctx, row.ID, perr.Error())
			continue
		}
		_ = s.sqlite.MarkOutboxProcessed(ctx, row.ID)
		processed++
	}
	return processed, nil
}
