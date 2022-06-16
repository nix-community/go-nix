package chunker

type Chunk struct {
	Offset uint64
	Data   []byte
}

// Chunker describes the interface that a given chunker needs to implement.
// Next() is periodically called until io.EOF is encountered.
// In case of no error, Next() returns a new chunk.

type Chunker interface {
	Next() (*Chunk, error)
}
