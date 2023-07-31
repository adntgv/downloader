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

const chunkSize = 1024 * 1024 * 10 // 10MB
type chunk struct {
	id    int
	start int64
	end   int64
}

func main() {
	fileURL := "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4"
	if err := download(fileURL); err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}
}

func download(fileURL string) error {
	url := fileURL

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	fileName := filepath.Base(url)
	out, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", fileName, err)
	}
	defer out.Close()

	fileSize := resp.ContentLength
	log.Printf("File size: %d bytes\n", fileSize)

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"

	if fileSize == -1 || !supportsRange {
		log.Println("File size unknown. Downloading with alternative parallelism...")
		return alternativeDownload(out, resp.Body)
	}

	log.Println("Starting parallel download...")

	return downloadInParallel(out, url, fileSize)
}

func alternativeDownload(outfile *os.File, in io.Reader) error {
	sharedChunkId := 0
	readerLock := sync.Mutex{}

	var wg sync.WaitGroup

	numWorkers := 5

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				readerLock.Lock()
				chunk := make([]byte, chunkSize)
				n, err := in.Read(chunk)
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

	log.Println("Download completed!")

	// assemble the chunks
	log.Println("Assembling chunks...")

	for i := 0; i < sharedChunkId; i++ {
		fileName := fmt.Sprintf("chunk-%d", i)
		in, err := os.Open(fileName)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return fmt.Errorf("error opening chunk file %s: %v", fileName, err)
		}
		defer in.Close()

		_, err = io.Copy(outfile, in)
		if err != nil {
			return fmt.Errorf("error writing chunk file %s to output file: %v", fileName, err)
		}

		err = os.Remove(fileName)
		if err != nil {
			return fmt.Errorf("error removing chunk file %s: %v", fileName, err)
		}
	}

	return nil
}

func downloadInParallel(out *os.File, url string, fileSize int64) error {
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
		if end >= fileSize {
			end = fileSize
		}

		log.Printf("Downloading chunk from offset %d to %d...\n", offset, end)
		chunkTasks <- chunk{id: chunkId, start: offset, end: end}
		chunkId++
		if end == fileSize {
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
			return fmt.Errorf("error opening chunk file %s: %v", fileName, err)
		}
		defer in.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("error writing chunk file %s to output file: %v", fileName, err)
		}

		err = os.Remove(fileName)
		if err != nil {
			return fmt.Errorf("error removing chunk file %s: %v", fileName, err)
		}
	}

	return nil
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
				log.Fatalf("Error creating chunk file %s: %v\n", fileName, err)
			}
			defer out.Close()

			log.Printf("Worker %d downloading chunk at offset %d...\n", workerId, chunk.start)
			resp, err := downloadChunk(url, chunk.start, chunk.end)
			if err != nil {
				log.Fatalf("Error downloading chunk at offset %d: %v\n", chunk.start, err)
			}
			defer resp.Body.Close()

			log.Printf("Writing chunk at offset %d to output file...\n", chunk.start)
			n, err := io.Copy(out, resp.Body)
			if err != nil {
				log.Fatalf("Error writing chunk at offset %d to output file: %v\n", chunk.start, err)
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
