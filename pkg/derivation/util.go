package derivation

// GetFixedOutput returns the fixed output if found, otherwise nil.
func (d *Derivation) GetFixedOutput() *Output {
	for _, o := range d.Outputs {
		if o.HashAlgorithm != "" {
			return o
		}
	}

	return nil
}
