package main

type Downloader interface {
	Download(url string, chunkPrefix string) error
}
