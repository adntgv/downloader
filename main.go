package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	fileURL := "https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4"
	chunkSize := 1024 * 1024 * 10 // 10MB
	numWorkers := 5
	chunkPrefix := "chunk-"
	fileName := filepath.Base(fileURL)

	if err := download(fileURL, chunkSize, numWorkers, chunkPrefix); err != nil {
		log.Fatalf("Error downloading file: %v\n", err)
	}

	log.Println("Download completed")

	err := assembleChunks(fileName, chunkPrefix)
	if err != nil {
		log.Fatalf("Error assembling chunks: %v\n", err)
	}
}

func download(fileURL string, chunkSize int, numWorkers int, chunkPrefix string) error {
	url := fileURL

	log.Printf("Downloading file from: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	supportsRange := resp.Header.Get("Accept-Ranges") == "bytes"

	downloader := newDownloader(chunkSize, numWorkers, resp.ContentLength, supportsRange)

	return downloader.Download(url, chunkPrefix)
}

func newDownloader(chunkSize int, numWorkers int, fileSize int64, supportsRange bool) Downloader {
	var downloader Downloader
	if fileSize == -1 || !supportsRange {
		log.Println("File size unknown. Downloading with alternative parallelism...")
		downloader = &AlternativeDownloader{
			ChunkSize:  chunkSize,
			NumWorkers: numWorkers,
		}
	} else {
		log.Println("File size known. Downloading with chunk parallelism...")
		downloader = &ParallelDownloader{
			ChunkSize:  chunkSize,
			NumWorkers: numWorkers,
			FileSize:   fileSize,
		}
	}

	return downloader
}

func assembleChunks(filename string, chunkPrefix string) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer out.Close()

	for i := 0; ; i++ {
		chunkFilename := fmt.Sprintf("%s%d", chunkPrefix, i)
		if _, err := os.Stat(chunkFilename); os.IsNotExist(err) {
			break
		}

		in, err := os.Open(chunkFilename)
		if err != nil {
			return fmt.Errorf("error opening chunk file %s: %v", chunkFilename, err)
		}

		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("error copying chunk file %s to output: %v", chunkFilename, err)
		}

		in.Close()

		err = os.Remove(chunkFilename)
		if err != nil {
			return fmt.Errorf("error removing chunk file %s: %v", chunkFilename, err)
		}
	}

	return nil
}
