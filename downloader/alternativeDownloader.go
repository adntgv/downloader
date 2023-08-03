package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type AlternativeDownloader struct {
	ChunkSize  int
	NumWorkers int
	Chunker    ChunkHandler
}

func NewAlternativeDownloader(chunkSize int, numWorkers int, chunker ChunkHandler) Downloader {
	return &AlternativeDownloader{
		ChunkSize:  chunkSize,
		NumWorkers: numWorkers,
		Chunker:    chunker,
	}
}

func (d *AlternativeDownloader) Download(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	sharedChunkId := 0
	readerLock := sync.Mutex{}
	numWorkers := d.NumWorkers
	chunkSize := d.ChunkSize

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go d.processChunks(resp, &wg, &sharedChunkId, &readerLock, chunkSize)
	}

	wg.Wait()

	return nil
}

func (d *AlternativeDownloader) processChunks(resp *http.Response, wg *sync.WaitGroup, sharedChunkId *int, readerLock *sync.Mutex, chunkSize int) {
	defer wg.Done()

	for {
		readerLock.Lock()
		chunk := make([]byte, chunkSize)
		n, err := resp.Body.Read(chunk)
		log.Printf("Read %d bytes, size %d\n", n, chunkSize)
		chunkId := *sharedChunkId
		*sharedChunkId++
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

		err = d.Chunker.Handle(chunkId, chunk)
		if err != nil {
			log.Fatalf("Error saving chunk %d to file: %v\n", chunkId, err)
		}
	}
}
