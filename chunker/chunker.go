package chunker

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type Chunk struct {
	Index int
	Data  []byte
}

func NewChunk(index int, data []byte) *Chunk {
	return &Chunk{
		Index: index,
		Data:  data,
	}
}

type Chunker interface {
	Handle(chunk *Chunk) error
	AssembleChunks(filename string) error
}

type DefaultChunker struct {
	ChunkPrefix string
}

func (c *DefaultChunker) Handle(chunk *Chunk) error {
	filename := fmt.Sprintf("%s%d", c.ChunkPrefix, chunk.Index)
	err := saveBytesToFile(filename, chunk.Data)
	if err != nil {
		return fmt.Errorf("error saving chunk %d to file: %v", chunk.Index, err)
	}

	return nil
}

func (c *DefaultChunker) AssembleChunks(filename string) error {

	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer out.Close()

	for i := 0; ; i++ {
		chunkFilename := fmt.Sprintf("%s%d", c.ChunkPrefix, i)
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

func NewChunker(
	chunkPrefix string) Chunker {
	return &DefaultChunker{
		ChunkPrefix: chunkPrefix,
	}
}

func saveBytesToFile(filename string, bz []byte) error {
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", filename, err)
	}
	defer out.Close()

	in := bytes.NewReader(bz)

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", filename, err)
	}

	return nil
}
