package downloader

import (
	"chunker"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type ChunkDownloader struct {
	ChunkSize  int
	NumWorkers int
	Chunker    chunker.Chunker
	done       chan struct{}
}

func NewChunkDownloader(chunkSize int, numWorkers int, chunker chunker.Chunker) Downloader {
	return &ChunkDownloader{
		ChunkSize:  chunkSize,
		NumWorkers: numWorkers,
		Chunker:    chunker,
		done:       make(chan struct{}),
	}
}

type IdBytes struct {
	Id    int
	Bytes []byte
}

func (d *ChunkDownloader) Download(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: %s", resp.Status)
	}

	chunkSize := d.ChunkSize
	numWorkers := d.NumWorkers
	chunkCh := make(chan IdBytes, numWorkers)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go d.processChunks(chunkCh, &wg)
	}

	chunkId := 0

	for {
		chunk := make([]byte, chunkSize)
		n, err := resp.Body.Read(chunk)
		if err != nil {
			if err == io.EOF {
				// Send the last chunk to the worker goroutines
				chunkCh <- IdBytes{Id: chunkId, Bytes: chunk[:n]}

				break
			}

			log.Fatalf("error reading chunk: %v", err)
		}

		if n > 0 {
			// Send the chunk to the worker goroutines
			chunkCh <- IdBytes{Id: chunkId, Bytes: chunk[:n]}
		}
		chunkId++
	}

	close(chunkCh) // Signal the worker goroutines that there are no more chunks to process
	wg.Wait()
	return nil
}

func (d *ChunkDownloader) processChunks(chunkCh <-chan IdBytes, wg *sync.WaitGroup) {
	defer wg.Done()
	for bz := range chunkCh {
		err := d.Chunker.Handle(bz.Id, bz.Bytes)
		if err != nil {
			log.Fatalf("Error saving chunk %d to file: %v\n", bz.Id, err)
		}
	}
}
