package downloader

type Downloader interface {
	Download(url string) error
}
