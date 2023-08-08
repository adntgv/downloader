package chunker

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const TMP_DIR = "/tmp/"

type DefaultChunker struct {
	ChunkPrefix string
	NextID      int
	ChunkChan   chan Chunk
	ChunkSize   int
	ChunkIDLock *sync.Mutex
}

func NewChunker(chunkPrefix string, chunkSize int) Chunker {
	return &DefaultChunker{
		ChunkPrefix: chunkPrefix,
		NextID:      0,
		ChunkChan:   make(chan Chunk),
		ChunkSize:   chunkSize,
		ChunkIDLock: &sync.Mutex{},
	}
}

func (c *DefaultChunker) NextChunkID() int {
	c.ChunkIDLock.Lock()
	id := c.NextID
	c.NextID++
	c.ChunkIDLock.Unlock()
	return id
}

func (c *DefaultChunker) Handle(id int, bz []byte) error {
	filename := c.getChunkFilename(id)
	return c.saveBytesToFile(filename, bz)
}

func (c *DefaultChunker) getChunkFilename(id int) string {
	return fmt.Sprintf("%s%s%d", TMP_DIR, c.ChunkPrefix, id)
}

func (c *DefaultChunker) saveBytesToFile(filename string, bz []byte) error {
	if _, err := os.Stat(filename); err == nil {
		// File exists
		return nil
	}

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

func (d *DefaultChunker) GenerateChunkTasks(fileSize int64) int {
	offset := int64(0)
	chunkId := 0

	chunkChan := d.GetChunkChannel()

	for {
		end := offset + int64(d.ChunkSize) - 1
		if end >= fileSize {
			end = fileSize
		}
		c := Chunk{ID: chunkId, Start: offset, End: end}
		log.Printf("Generated chunk %v\n", c)
		chunkChan <- c
		chunkId++
		if end == fileSize {
			break
		}

		offset = end + 1
	}

	return chunkId
}
