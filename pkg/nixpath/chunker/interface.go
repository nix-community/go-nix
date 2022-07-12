package chunker

type Chunk struct {
	Offset uint64
	Data   []byte
}

// Chunker describes the interface that a given chunker needs to implement.
// Next() is periodically called until io.EOF is encountered.
// In case of no error, Next() returns a new chunk.

// TODO: is this interface the right one, or should we add initialization
// to the interface? Look at how it's used in pkg/store/import.go

type Chunker interface {
	Next() (*Chunk, error)
}
