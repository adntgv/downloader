package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

type AlternativeDownloader struct {
	ChunkSize  int
	NumWorkers int
}

func (d *AlternativeDownloader) Download(url string, chunkPrefix string) error {
	sharedChunkId := 0
	readerLock := sync.Mutex{}

	var wg sync.WaitGroup

	numWorkers := d.NumWorkers
	chunkSize := d.ChunkSize

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				readerLock.Lock()
				chunk := make([]byte, chunkSize)
				n, err := resp.Body.Read(chunk)
				chunkId := sharedChunkId
				sharedChunkId++
				readerLock.Unlock()

				if err != nil {
					if err == io.EOF {
						return
					}
					log.Fatalf("Error reading chunk %d: %v\n", chunkId, err)
				}

				if n == 0 {
					return
				}

				if n < chunkSize {
					chunk = chunk[:n]
				}

				chunkFileName := fmt.Sprintf("chunk-%d", chunkId)
				out, err := os.Create(chunkFileName)
				if err != nil {
					log.Fatalf("Error creating chunk file %s: %v\n", chunkFileName, err)
				}

				_, err = out.Write(chunk[:n])
				if err != nil {
					log.Fatalf("Error writing chunk %d to file: %v\n", chunkId, err)
				}
			}
		}()
	}

	wg.Wait()

	return nil
}
