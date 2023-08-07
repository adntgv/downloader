package downloader

import (
	"chunker"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type ParallelDownloader struct {
	ChunkSize  int
	NumWorkers int
	FileSize   int64
	Chunker    chunker.Chunker
	wg         *sync.WaitGroup
}

func NewParallelDownloader(
	chunkSize int,
	numWorkers int,
	fileSize int64,
	chunker chunker.Chunker,
	wg *sync.WaitGroup,
) Downloader {
	return &ParallelDownloader{
		ChunkSize:  chunkSize,
		NumWorkers: numWorkers,
		FileSize:   fileSize,
		Chunker:    chunker,
		wg:         wg,
	}
}

func (d *ParallelDownloader) Download(url string) error {
	// Create channel to send chunk tasks to workers
	chunkTasks := d.Chunker.GetChunkChannel()

	// Launch worker goroutines
	for i := 0; i < d.NumWorkers; i++ {
		go d.chunkDownloader(i, chunkTasks, url, d.wg)
	}

	return nil
}

func (d *ParallelDownloader) chunkDownloader(workerId int, chunkTasks <-chan chunker.Chunk, url string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for chunk := range chunkTasks {
		d.downloadAndSave(workerId, chunk, url)
	}
}

func (d *ParallelDownloader) downloadAndSave(workerId int, chunk chunker.Chunk, url string) {
	log.Printf("Worker %d downloading chunk %d\n", workerId, chunk.ID)
	resp, err := d.downloadChunk(url, chunk.Start, chunk.End)
	if err != nil {
		log.Fatalf("Error downloading chunk at offset %d: %v\n", chunk.Start, err)
	}
	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading chunk %d: %v\n", chunk.ID, err)
	}

	err = d.Chunker.Handle(chunk.ID, bz)
	if err != nil {
		log.Fatalf("Error saving chunk %d to file: %v\n", chunk.ID, err)
	}
}

func (d *ParallelDownloader) downloadChunk(url string, start, end int64) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Add("Range", rangeHeader)

	return http.DefaultClient.Do(req)
}
