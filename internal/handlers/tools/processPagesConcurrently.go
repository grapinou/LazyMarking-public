package tools

import (
	"context"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
)

func ProcessPagesConcurrently(pages []string, tempDir string, queries *db.Queries, ctx context.Context, jobDBID int64, userID int64) ([]config.QrCodeInfo, []string, error) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Sémaphore: max 5 goroutines
	var mu sync.Mutex             // protège l’accès aux slices partagées

	var qrDatas []config.QrCodeInfo
	var qrNotDetected []string

	var firstErr error
	errOnce := sync.Once{} // pour ne garder que la première erreur critique

	for _, page := range pages {
		wg.Add(1)
		page := page // éviter le piège de la closure

		go func() {
			defer wg.Done()

			// Prend un "ticket" dans la sémaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			pdf := filepath.Base(page)
			name := strings.TrimSuffix(pdf, filepath.Ext(page)) + ".png"
			png, err := ConvertPdfToPng(tempDir, pdf, "")
			if err != nil {
				errOnce.Do(func() { firstErr = err })
				return
			}
			pngPath := filepath.Join(tempDir, png)
			data, err := QrReader(pngPath)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				log.Printf("file : %s, qr not detected : %v", pngPath, err)
				qrNotDetected = append(qrNotDetected, png)
				return
			}

			var info config.QrCodeInfo
			if err := json.Unmarshal([]byte(data), &info); err != nil {
				log.Printf("Unmarshal failed for %s: %v", pngPath, err)
				qrNotDetected = append(qrNotDetected, png)
				return
			}
			info.PageName = name
			qrDatas = append(qrDatas, info)

			if err := queries.UpdateMarkingJobPageDone(ctx, db.UpdateMarkingJobPageDoneParams{
				ID:     jobDBID,
				UserID: userID,
			}); err != nil {
				log.Printf("From ProcessPagesConcurrently -> queries.UpdateMarkingJobPageDone DB error : %v", err)
				errOnce.Do(func() { firstErr = err }) // capture la première erreur critique
			}
		}()
	}

	wg.Wait()
	return qrDatas, qrNotDetected, firstErr
}
