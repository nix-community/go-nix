package derivation

// Store describes the interface a Derivation store needs to implement
// to be used from here.
// Note we use pointers to Derivation structs here, so be careful modifying these.
// Look in the store/ subfolder for implementations.
type Store interface {
	// Get retrieves a derivation by drv path.
	// The second return argument specifies if the derivation could be found,
	// similar to how acessing from a map works.
	Get(string) (*Derivation, error)

	// GetSubstitutionHash produces a hex-encoded hash of the current derivation.
	// It recursively does this for all Input Derivations, so implementations might
	// want to cache these results.
	GetSubstitutionHash(string) (string, error)
}

// StorePut describes the interface a Derivation store implements,
// that also allows inserting Derivation structs.
type StorePut interface {
	Store
	// Put inserts a new Derivation into the Derivation Store.
	// All referred derivation paths should have been Put() before.
	// The resulting derivation path is returned, or an error.
	Put(*Derivation) (string, error)
}
