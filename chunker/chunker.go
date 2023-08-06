package chunker

type Chunk struct {
	ID    int
	Start int64
	End   int64
}

type Chunker interface {
	Handle(id int, bz []byte) error
	AssembleChunks(filename string) error
	NextChunkID() int
	GetChunkChannel() chan Chunk
}
