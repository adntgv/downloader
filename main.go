package main

import (
	"chunker"
	"downloader"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	fileURLFlag          = flag.String("url", "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4", "URL of the file to download")
	chunkSizeFlag        = flag.Int("chunkSize", 1024*1024*10, "Size of each chunk in bytes")
	numWorkersFlag       = flag.Int("numWorkers", 5, "Number of workers to use")
	chunkPrefixFlag      = flag.String("chunkPrefix", "chunk-", "Prefix of the chunk files")
	fileNameFlag         = flag.String("fileName", "sample_1280x720_surfing_with_audio.mp4", "Name of the file to download")
	forceAlternativeFlag = flag.Bool("forceAlternative", false, "Force the alternative downloader to be used")
)

func main() {
	flag.Parse()

	fileURL := *fileURLFlag
	chunkSize := *chunkSizeFlag
	numWorkers := *numWorkersFlag
	chunkPrefix := *chunkPrefixFlag
	fileName := *fileNameFlag
	forceAlternative := *forceAlternativeFlag

	chunker := chunker.NewChunker(chunkPrefix)

	if err := download(fileURL, chunkSize, numWorkers, chunker, forceAlternative); err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}

	log.Println("Download completed")

	err := chunker.AssembleChunks(fileName)
	if err != nil {
		log.Fatalf("Error assembling chunks: %v\n", err)
	}
}

func download(fileURL string, chunkSize int, numWorkers int, chunker chunker.Chunker, forceAlternative bool) error {
	url := fileURL

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"

	downloader := newDownloader(chunkSize, numWorkers, resp.ContentLength, supportsRange, chunker, forceAlternative)

	return downloader.Download(url)
}

func newDownloader(chunkSize int, numWorkers int, fileSize int64, supportsRange bool, chunker chunker.Chunker, forceAlternative bool) downloader.Downloader {
	var d downloader.Downloader
	if fileSize == -1 || !supportsRange || forceAlternative {
		log.Println("File size unknown. Downloading with alternative parallelism...")
		d = downloader.NewAlternativeDownloader(
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
		)
	}

	return d
}
