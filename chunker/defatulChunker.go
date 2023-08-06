package chunker

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
)

type DefaultChunker struct {
	ChunkPrefix string
	NextID      int
	ChunkChan   chan Chunk
}

func NewChunker(chunkPrefix string) Chunker {
	return &DefaultChunker{
		ChunkPrefix: chunkPrefix,
		NextID:      0,
		ChunkChan:   make(chan Chunk),
	}
}

func (c *DefaultChunker) NextChunkID() int {
	id := c.NextID
	c.NextID++
	return id
}

func (c *DefaultChunker) Handle(id int, bz []byte) error {
	filename := c.getChunkFilename(id)
	return c.saveBytesToFile(filename, bz)
}

func (c *DefaultChunker) getChunkFilename(id int) string {
	return fmt.Sprintf("%s%d", c.ChunkPrefix, id)
}

func (c *DefaultChunker) saveBytesToFile(filename string, bz []byte) error {
	log.Printf("Saving chunk %s\n", filename)
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", filename, err)
	}
	defer out.Close()

	_, err = io.Copy(out, bytes.NewReader(bz))
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", filename, err)
	}

	return nil
}

func (c *DefaultChunker) AssembleChunks(filename string) error {
	log.Printf("Assembling chunks into %s\n", filename)
	out, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file %s: %w", filename, err)
	}
	defer out.Close()

	for i := 0; ; i++ {
		chunkFilename := c.getChunkFilename(i)
		if _, err := os.Stat(chunkFilename); os.IsNotExist(err) {
			break
		}

		in, err := os.Open(chunkFilename)
		if err != nil {
			return fmt.Errorf("error opening chunk file %s: %w", chunkFilename, err)
		}
		defer in.Close()

		_, err = io.Copy(out, in)
		if err != nil {
			return fmt.Errorf("error copying chunk file %s to output: %w", chunkFilename, err)
		}

		err = os.Remove(chunkFilename)
		if err != nil {
			return fmt.Errorf("error removing chunk file %s: %w", chunkFilename, err)
		}
	}

	return nil
}

func (c *DefaultChunker) GetChunkChannel() chan Chunk {
	return c.ChunkChan
}
