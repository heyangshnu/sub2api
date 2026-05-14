package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"sub2api-go/internal/model"
	"sub2api-go/internal/store"
)

// Exercises concurrent PreDeduct + Finalize on MemoryStore (mutex correctness).
func TestMemoryStoreConcurrentPreDeductFinalize(t *testing.T) {
	ctx := context.Background()
	s := store.NewMemoryStore()

	raw, key, err := s.CreateKey(ctx, "u1", "t", 1000, 60)
	if err != nil || key == nil {
		t.Fatalf("CreateKey: %v", err)
	}
	_ = raw

	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			est := 0.01
			if err := s.PreDeduct(ctx, key.KeyHash, est); err != nil {
				return
			}
			u := model.Usage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20}
			act := 0.005
			_ = s.FinalizeDeduct(ctx, key.KeyHash, est, act, u, "gpt-4o-mini", "req_"+time.Now().Format("150405.000000000"))
		}()
	}
	wg.Wait()

	bal, err := s.GetBalance(ctx, key.KeyHash)
	if err != nil {
		t.Fatal(err)
	}
	if bal < 0 {
		t.Fatalf("negative balance: %f", bal)
	}
}
