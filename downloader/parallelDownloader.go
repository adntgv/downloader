package downloader

import (
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
	Chunker    ChunkHandler
}

func NewParallelDownloader(
	chunkSize int,
	numWorkers int,
	fileSize int64,
	chunker ChunkHandler) Downloader {
	return &ParallelDownloader{
		ChunkSize:  chunkSize,
		NumWorkers: numWorkers,
		FileSize:   fileSize,
		Chunker:    chunker,
	}
}

type chunk struct {
	id    int
	start int64
	end   int64
}

func (d *ParallelDownloader) Download(url string) error {
	var wg sync.WaitGroup

	// Create channel to send chunk tasks to workers
	chunkTasks := make(chan chunk)

	// Determine the number of worker goroutines (adjust as needed)
	numWorkers := 5

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		go d.chunkDownloader(i, chunkTasks, url, &wg)
	}

	// Generate chunk tasks
	d.generateChunkTasks(d.FileSize, chunkTasks, d.ChunkSize)

	wg.Wait()

	return nil
}

func (d *ParallelDownloader) generateChunkTasks(fileSize int64, chunkTasks chan<- chunk, chunkSize int) int {
	offset := int64(0)
	chunkId := 0

	for {
		end := offset + int64(chunkSize) - 1
		if end >= fileSize {
			end = fileSize
		}
		c := chunk{id: chunkId, start: offset, end: end}
		log.Printf("Generated chunk %v\n", c)
		chunkTasks <- c
		chunkId++
		if end == fileSize {
			break
		}

		offset = end + 1
	}

	close(chunkTasks)

	return chunkId
}

func (d *ParallelDownloader) chunkDownloader(workerId int, chunkTasks <-chan chunk, url string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for chunk := range chunkTasks {
		d.downloadAndSave(workerId, chunk, url)
	}
}

func (d *ParallelDownloader) downloadAndSave(workerId int, chunk chunk, url string) {
	log.Printf("Worker %d downloading chunk %d\n", workerId, chunk.id)
	resp, err := d.downloadChunk(url, chunk.start, chunk.end)
	if err != nil {
		log.Fatalf("Error downloading chunk at offset %d: %v\n", chunk.start, err)
	}
	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading chunk %d: %v\n", chunk.id, err)
	}

	err = d.Chunker.Handle(chunk.id, bz)
	if err != nil {
		log.Fatalf("Error saving chunk %d to file: %v\n", chunk.id, err)
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
