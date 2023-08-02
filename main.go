package main

import (
	"chunker"
	"downloader"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	fileURL := "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4"
	chunkSize := 1024 * 1024 * 10 // 10MB
	numWorkers := 5
	chunkPrefix := "chunk-"
	fileName := filepath.Base(fileURL)

	chunker := chunker.NewChunker(chunkPrefix)

	if err := download(fileURL, chunkSize, numWorkers, chunker); err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}

	log.Println("Download completed")

	err := chunker.AssembleChunks(fileName)
	if err != nil {
		log.Fatalf("Error assembling chunks: %v\n", err)
	}
}

func download(fileURL string, chunkSize int, numWorkers int, chunker chunker.Chunker) error {
	url := fileURL

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"

	downloader := newDownloader(chunkSize, numWorkers, resp.ContentLength, supportsRange, chunker)

	return downloader.Download(url)
}

func newDownloader(chunkSize int, numWorkers int, fileSize int64, supportsRange bool, chunker chunker.Chunker) downloader.Downloader {
	var d downloader.Downloader
	if fileSize == -1 || !supportsRange {
		log.Println("File size unknown. Downloading with alternative parallelism...")
		d = &downloader.AlternativeDownloader{
			ChunkSize:  chunkSize,
			NumWorkers: numWorkers,
			Chunker:    chunker,
		}
	} else {
		log.Println("File size known. Downloading with chunk parallelism...")
		d = &downloader.ParallelDownloader{
			ChunkSize:  chunkSize,
			NumWorkers: numWorkers,
			FileSize:   fileSize,
			Chunker:    chunker,
		}
	}

	return d
}
