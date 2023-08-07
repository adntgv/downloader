package main

import (
	"chunker"
	"downloader"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"sync"
)

var (
	fileURLFlag          = flag.String("url", "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4", "URL of the file to download")
	chunkSizeFlag        = flag.Int("chunkSize", 1024*1024*10, "Size of each chunk in bytes")
	numWorkersFlag       = flag.Int("numWorkers", 5, "Number of workers to use")
	chunkPrefixFlag      = flag.String("chunkPrefix", "chunk-", "Prefix of the chunk files")
	forceAlternativeFlag = flag.Bool("forceAlternative", false, "Force the alternative downloader to be used")
)

func main() {
	flag.Parse()

	fileURL := *fileURLFlag
	chunkSize := *chunkSizeFlag
	numWorkers := *numWorkersFlag
	chunkPrefix := *chunkPrefixFlag
	forceAlternative := *forceAlternativeFlag
	fileName := path.Base(fileURL)
	c := chunker.NewChunker(chunkPrefix, chunkSize)
	wg := &sync.WaitGroup{}

	if err := download(fileURL, chunkSize, numWorkers, c, forceAlternative, wg); err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}

	wg.Wait()

	log.Println("Download completed")

	if err := c.AssembleChunks(fileName); err != nil {
		log.Fatalf("Error assembling chunks: %v\n", err)
	}
}

func download(fileURL string, chunkSize int, numWorkers int, c chunker.Chunker, forceAlternative bool, wg *sync.WaitGroup) error {
	url := fileURL

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"
	fileSize := resp.ContentLength

	// Generate chunk tasks
	wg.Add(1)
	go func(c chunker.Chunker, wg *sync.WaitGroup, fileSize int64) {
		defer wg.Done()
		c.GenerateChunkTasks(fileSize)

		close(c.GetChunkChannel())
	}(c, wg, fileSize)

	downloader := newDownloader(chunkSize, numWorkers, resp.ContentLength, supportsRange, c, forceAlternative, wg)

	return downloader.Download(url)
}

func newDownloader(chunkSize int, numWorkers int, fileSize int64, supportsRange bool, chunker chunker.Chunker, forceAlternative bool, wg *sync.WaitGroup) downloader.Downloader {
	var d downloader.Downloader
	if fileSize == -1 || !supportsRange || forceAlternative {
		log.Println("File size unknown. Downloading with alternative parallelism...")
		d = downloader.NewChunkDownloader(
			chunkSize,
			numWorkers,
			chunker,
		)
	} else {
		log.Println("File size known. Downloading with range parallelism...")
		d = downloader.NewParallelDownloader(
			chunkSize,
			numWorkers,
			fileSize,
			chunker,
			wg,
		)
	}

	return d
}
