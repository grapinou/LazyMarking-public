package tools

import (
	"encoding/json"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grapinou/LazyMarking/internal/config"
)

func ProcessPagesConcurrently(pages []string, tempDir string) ([]config.QrCodeInfo, []string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // Sémaphore: max 5 goroutines
	var mu sync.Mutex             // protège l’accès aux slices partagées

	var qrDatas []config.QrCodeInfo
	var qrNotDetected []string

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
			png := ConvertPdfToPng(tempDir, pdf, "")
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
		}()
	}

	wg.Wait()
	return qrDatas, qrNotDetected
}
