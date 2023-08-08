package main

import (
	"bufio"
	"chunker"
	"downloader"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
)

var (
	chunkSizeFlag        = flag.Int("chunkSize", 1024*1024*10, "Size of each chunk in bytes")
	numWorkersFlag       = flag.Int("numWorkers", 5, "Number of workers to use")
	chunkPrefixFlag      = flag.String("chunkPrefix", "chunk-", "Prefix of the chunk files")
	forceAlternativeFlag = flag.Bool("forceAlternative", false, "Force the alternative downloader to be used")
	urlsFileFlag         = flag.String("urlsFile", "urls.txt", "File containing the urls to download")
)

func main() {
	flag.Parse()

	chunkSize := *chunkSizeFlag
	numWorkers := *numWorkersFlag
	chunkPrefix := *chunkPrefixFlag
	forceAlternative := *forceAlternativeFlag
	urlsFile := *urlsFileFlag

	urls, err := readURLs(urlsFile)
	if err != nil {
		log.Fatalf("Error reading urls file: %v\n", err)
	}

	c := chunker.NewChunker(chunkPrefix, chunkSize)
	wg := &sync.WaitGroup{}

	fileName, err := download(urls, chunkSize, numWorkers, c, forceAlternative, wg)
	if err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}

	wg.Wait()

	log.Println("Download completed")

	if err := c.AssembleChunks(fileName); err != nil {
		log.Fatalf("Error assembling chunks: %v\n", err)
	}
}

func download(urls []string, chunkSize int, numWorkers int, c chunker.Chunker, forceAlternative bool, wg *sync.WaitGroup) (string, error) {
	url := urls[0]
	fileName := path.Base(url)

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Head(url)
	if err != nil {
		return "", fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"
	fileSize := resp.ContentLength

	downloader := newDownloader(chunkSize, numWorkers, fileSize, supportsRange, c, forceAlternative, wg)

	// Generate chunk tasks
	wg.Add(1)
	go func(c chunker.Chunker, wg *sync.WaitGroup, fileSize int64) {
		defer wg.Done()
		c.GenerateChunkTasks(fileSize)

		close(c.GetChunkChannel())
	}(c, wg, fileSize)

	for _, url := range urls {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			downloader.Download(url)
		}(url)
	}

	return fileName, nil
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

func readURLs(urlsFile string) ([]string, error) {
	urls := []string{}

	file, err := os.Open(urlsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := scanner.Text()

		if url == "" {
			continue
		}

		urls = append(urls, url)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no urls found in file")
	}

	return urls, nil
}
