package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

type chunk struct {
	id    int
	start int64
	end   int64
}

type ParallelDownloader struct {
	ChunkSize  int
	NumWorkers int
	FileSize   int64
}

func (d *ParallelDownloader) Download(url string, chunkPrefix string) error {
	var wg sync.WaitGroup
	chunkTasks := make(chan chunk) // Channel to send chunk tasks to workers
	done := make(chan struct{})    // Channel to notify workers to stop

	// Determine the number of worker goroutines (adjust as needed)
	numWorkers := 5

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		go chunkDownloader(i, chunkTasks, url, &wg, done)
	}

	// Generate chunk tasks
	generateChunkTasks(d.FileSize, chunkTasks, d.ChunkSize)

	close(chunkTasks)

	wg.Wait()

	// Notify workers to stop
	close(done)

	return nil
}

func generateChunkTasks(fileSize int64, chunkTasks chan<- chunk, chunkSize int) int {
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

func chunkDownloader(workerId int, chunkTasks <-chan chunk, url string, wg *sync.WaitGroup, done <-chan struct{}) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case chunk, ok := <-chunkTasks:
			if !ok {
				return // No more tasks, worker can exit
			}
			downloadAndSave(workerId, chunk, url)
		case <-done:
			return // Exit worker goroutine when done channel is closed
		}
	}
}

func downloadAndSave(workerId int, chunk chunk, url string) {
	resp, err := downloadChunk(url, chunk.start, chunk.end)
	if err != nil {
		log.Fatalf("Error downloading chunk at offset %d: %v\n", chunk.start, err)
	}
	defer resp.Body.Close()

	err = saveByteStreamToFile(fmt.Sprintf("chunk-%d", chunk.id), resp.Body)
	if err != nil {
		log.Fatalf("Error saving chunk %d to file: %v\n", chunk.id, err)
	}
}

func downloadChunk(url string, start, end int64) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Add("Range", rangeHeader)

	return http.DefaultClient.Do(req)
}

func saveByteStreamToFile(filename string, in io.Reader) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", filename, err)
	}

	return nil
}
