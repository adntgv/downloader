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
}

type chunk struct {
	id    int
	start int64
	end   int64
}

func (d *ParallelDownloader) Download(url string) error {
	var wg sync.WaitGroup
	done := make(chan struct{}) // Channel to notify workers to stop

	// Create channel to send chunk tasks to workers
	chunkTasks := make(chan chunk)

	// Determine the number of worker goroutines (adjust as needed)
	numWorkers := 5

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		go d.chunkDownloader(i, chunkTasks, url, &wg, done)
	}

	// Generate chunk tasks
	d.generateChunkTasks(d.FileSize, chunkTasks, d.ChunkSize)

	wg.Wait()

	// Notify workers to stop
	close(done)

	return nil
}

func (d *ParallelDownloader) generateChunkTasks(fileSize int64, chunkTasks chan<- chunk, chunkSize int) int {
	offset := int64(0)
	chunkId := 0

	for {
		end := offset + int64(chunkSize)
		if end >= fileSize {
			end = fileSize
		}
		chunkTasks <- chunk{id: chunkId, start: offset, end: end}
		chunkId++
		if end == fileSize {
			break
		}

		offset = end
	}

	return chunkId
}

func (d *ParallelDownloader) chunkDownloader(workerId int, chunkTasks <-chan chunk, url string, wg *sync.WaitGroup, done <-chan struct{}) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case chunk, ok := <-chunkTasks:
			if !ok {
				return // No more tasks, worker can exit
			}
			d.downloadAndSave(workerId, chunk, url)
		case <-done:
			return // Exit worker goroutine when done channel is closed
		}
	}
}

func (d *ParallelDownloader) downloadAndSave(workerId int, chunk chunk, url string) {
	resp, err := d.downloadChunk(url, chunk.start, chunk.end)
	if err != nil {
		log.Fatalf("Error downloading chunk at offset %d: %v\n", chunk.start, err)
	}
	defer resp.Body.Close()

	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading chunk %d: %v\n", chunk.id, err)
	}

	c := chunker.NewChunk(chunk.id, bz)

	err = d.Chunker.Handle(c)
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
