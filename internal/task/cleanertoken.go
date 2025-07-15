package task

import (
	"context"
	"log"
	"time"

	"github.com/grapinou/LazyMarking/internal/db"
)

func StartTokenCleaner(ctx context.Context, queries *db.Queries, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				log.Println("[TokenCleaner] Running cleanup...")
				if err := queries.DeleteExpiredResetTokens(ctx); err != nil {
					log.Printf("[TokenCleaner] Failed to clean tokens: %v", err)
				}
			case <-ctx.Done():
				log.Println("[TokenCleaner] Shutting down.")
				return
			}
		}
	}()
}
