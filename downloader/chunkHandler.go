package downloader

type ChunkHandler interface {
	Handle(id int, bz []byte) error
	NextChunkID() int
}
