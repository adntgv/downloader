package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

const chunkSize = 1024 * 1024 * 5 // 5MB

func main() {
	fileURL := "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4"
	downloadFile(fileURL)
}

type chunk struct {
	id    int
	start int64
	end   int64
}

func downloadFile(url string) {
	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Error while downloading the file: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fileSize := resp.ContentLength
	log.Printf("File size: %d bytes\n", fileSize)

	if fileSize == -1 {
		log.Println("File size unknown. Downloading in chunks with parallelism...")
	}

	fileName := filepath.Base(url)
	out, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Error creating the output file: %v\n", err)
		return
	}
	defer out.Close()

	log.Println("Starting download...")

	var wg sync.WaitGroup
	chunkTasks := make(chan chunk) // Channel to send chunk tasks to workers
	done := make(chan struct{})    // Channel to notify workers to stop

	// Determine the number of worker goroutines (adjust as needed)
	numWorkers := 5

	// Launch worker goroutines
	for i := 0; i < numWorkers; i++ {
		go chunkDownloader(i, chunkTasks, url, &wg, done)
	}

	offset := int64(0)
	chunkId := 0
	for {
		end := offset + chunkSize
		if fileSize != -1 && end >= fileSize {
			end = fileSize
		}

		log.Printf("Downloading chunk from offset %d to %d...\n", offset, end)
		chunkTasks <- chunk{id: chunkId, start: offset, end: end}
		chunkId++
		if fileSize == -1 || end == fileSize {
			break
		}

		offset = end
	}

	close(chunkTasks)

	wg.Wait()

	// Notify workers to stop
	close(done)

	log.Println("Download completed!")

	// assemble the chunks
	log.Println("Assembling chunks...")

	for i := 0; i < chunkId; i++ {
		fileName := fmt.Sprintf("chunk-%d", i)
		in, err := os.Open(fileName)
		if err != nil {
			log.Printf("Error opening chunk file %s: %v\n", fileName, err)
			continue
		}
		defer in.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			log.Printf("Error writing chunk file %s to output file: %v\n", fileName, err)
		} else {
			log.Printf("Chunk file %s written to output file\n", fileName)
		}

		err = os.Remove(fileName)
		if err != nil {
			log.Printf("Error removing chunk file %s: %v\n", fileName, err)
		} else {
			log.Printf("Chunk file %s removed\n", fileName)
		}
	}
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

			fileName := fmt.Sprintf("chunk-%d", chunk.id)
			out, err := os.Create(fileName)
			if err != nil {
				log.Printf("Error creating chunk file %s: %v\n", fileName, err)
				continue
			}
			defer out.Close()

			log.Printf("Worker %d downloading chunk at offset %d...\n", workerId, chunk.start)
			resp, err := downloadChunk(url, chunk.start, chunk.end)
			if err != nil {
				log.Printf("Error downloading chunk at offset %d: %v\n", chunk.start, err)
				continue
			}
			defer resp.Body.Close()

			log.Printf("Writing chunk at offset %d to output file...\n", chunk.start)
			n, err := io.Copy(out, resp.Body)
			if err != nil {
				log.Printf("Error writing chunk at offset %d to output file: %v\n", chunk.start, err)
			} else {
				log.Printf("Chunk at offset %d downloaded and written to output file\n", chunk.start)
			}
			log.Printf("Chunk at offset %d size: %d bytes\n", chunk.start, n)
		case <-done:
			return // Exit worker goroutine when done channel is closed
		}
	}
}

func downloadChunk(url string, start, end int64) (*http.Response, error) {
	log.Printf("Downloading chunk from %d to %d...\n", start, end)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)
	req.Header.Add("Range", rangeHeader)

	return http.DefaultClient.Do(req)
}
