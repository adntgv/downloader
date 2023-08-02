package chunker

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type Chunker interface {
	Handle(id int, bz []byte) error
	AssembleChunks(filename string) error
}

type DefaultChunker struct {
	ChunkPrefix string
}

func NewChunker(
	chunkPrefix string) Chunker {
	return &DefaultChunker{
		ChunkPrefix: chunkPrefix,
	}
}

func (c *DefaultChunker) Handle(id int, bz []byte) error {
	filename := fmt.Sprintf("%s%d", c.ChunkPrefix, id)
	err := c.saveBytesToFile(filename, bz)
	if err != nil {
		return fmt.Errorf("error saving chunk %d to file: %v", id, err)
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

func (c *DefaultChunker) saveBytesToFile(filename string, bz []byte) error {
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
