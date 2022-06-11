package derivation

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

var (
	derivationPrefix  = []byte("Derive") // nolint:gochecknoglobals
	errArrayNotClosed = fmt.Errorf("array not closed")
)

// ReadDerivation parses a Derivation in ATerm format and returns the Derivation struct,
// or an error in case any parsing error occurs, or some of the fields would be illegal.
func ReadDerivation(reader io.Reader) (*Derivation, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	drv, err := parseDerivation(bytes)
	if err != nil {
		return nil, err
	}

	return drv, drv.Validate()
}

// parseDerivation provides a derivation parser that works without any memory allocations.
// It does so by walking the byte slice recursively and calling a callback for every array item found
// with the array item sub-sliced from the passed slice.
// During parsing, it checks for some invalid inputs (e.g. maps in the wrong order) that won't be
// recognizable in the returned struct.
// Other checks are handled by Derivation.Validate(),
// which is called by ReadDerivation() after parseDerivation().
func parseDerivation(derivationBytes []byte) (*Derivation, error) {
	if len(derivationBytes) < 8 {
		return nil, fmt.Errorf("input too short to be a valid derivation")
	}

	if !bytes.Equal(derivationBytes[:6], derivationPrefix) {
		return nil, fmt.Errorf("missing derivation prefix")
	}

	drv := &Derivation{}

	// https://github.com/golang/go/issues/37711
	drv.InputSources = []string{}
	drv.Arguments = []string{}

	err := arrayEach(derivationBytes[6:], func(value []byte, index int) error {
		var err error

		switch index {
		case 0: // Outputs
			drv.Outputs = make(map[string]*Output)
			// Outputs are always lexicographically sorted by their name.
			// keep track of the previous path read (if any), so we detect
			// invalid encodings.
			prevOutputName := ""
			err = arrayEach(value, func(value []byte, index int) error {
				output := &Output{}
				outputName := ""

				// Get every output field
				err := arrayEach(value, func(value []byte, index int) error {
					var err error
					switch index {
					case 0:
						outputName, err = unquote(value)
						if err != nil {
							return err
						}
						if outputName <= prevOutputName {
							return fmt.Errorf("invalid output order, %s <= %s", outputName, prevOutputName)
						}
					case 1:
						output.Path, err = unquote(value)
						if err != nil {
							return err
						}
					case 2:
						output.HashAlgorithm, err = unquote(value)
						if err != nil {
							return err
						}
					case 3:
						output.Hash, err = unquote(value)
						if err != nil {
							return err
						}
					default:
						return fmt.Errorf("unhandled output index: %d", index)
					}

					return nil
				})
				if err != nil {
					return err
				}

				if outputName == "" {
					return fmt.Errorf("output name for %s may not be empty", output.Path)
				}
				drv.Outputs[outputName] = output
				prevOutputName = outputName

				return nil
			})

		case 1: // InputDerivations
			drv.InputDerivations = make(map[string][]string)
			// InputDerivations are always lexicographically sorted by their path
			prevInputDrvPath := ""
			err = arrayEach(value, func(value []byte, index int) error {
				inputDrvPath := ""
				inputDrvNames := []string{}

				err := arrayEach(value, func(value []byte, index int) error {
					var err error
					switch index {
					case 0:
						inputDrvPath, err = unquote(value)
						if err != nil {
							return err
						}
						if inputDrvPath <= prevInputDrvPath {
							return fmt.Errorf("invalid input derivation order: %s <= %s", inputDrvPath, prevInputDrvPath)
						}

					case 1:
						err := arrayEach(value, func(value []byte, index int) error {
							unquoted, err := unquote(value)
							if err != nil {
								return err
							}
							inputDrvNames = append(inputDrvNames, unquoted)

							return nil
						})
						if err != nil {
							return err
						}

					default:
						return fmt.Errorf("unhandled input derivation index: %d", index)
					}

					return nil
				})
				if err != nil {
					return err
				}

				drv.InputDerivations[inputDrvPath] = inputDrvNames
				prevInputDrvPath = inputDrvPath

				return nil
			})

		case 2: // InputSources
			err = arrayEach(value, func(value []byte, index int) error {
				unquoted, err := unquote(value)
				if err != nil {
					return err
				}
				drv.InputSources = append(drv.InputSources, unquoted)

				return nil
			})

		case 3: // Platform
			drv.Platform, err = unquote(value)

		case 4: // Builder
			drv.Builder, err = unquote(value)

		case 5: // Arguments
			err = arrayEach(value, func(value []byte, index int) error {
				unquoted, err := unquote(value)
				if err != nil {
					return err
				}
				drv.Arguments = append(drv.Arguments, unquoted)

				return nil
			})

		case 6: // Env
			drv.Env = make(map[string]string)
			prevEnvKey := ""
			err = arrayEach(value, func(value []byte, index int) error {
				envValue := ""
				envKey := ""

				// For every field
				err := arrayEach(value, func(value []byte, index int) error {
					var err error
					switch index {
					case 0:
						envKey, err = unquote(value)
						if err != nil {
							return err
						}
						if envKey <= prevEnvKey {
							return fmt.Errorf("invalid env var order: %s <= %s", envKey, prevEnvKey)
						}
					case 1:
						envValue, err = unquote(value)
						if err != nil {
							return err
						}
					default:
						return fmt.Errorf("unhandled env var index: %d", index)
					}

					return nil
				})
				if err != nil {
					return err
				}

				drv.Env[envKey] = envValue
				prevEnvKey = envKey

				return err
			})

		default:
			return fmt.Errorf("unhandled derivation index: %d", index)
		}

		return err
	})
	if err != nil {
		return nil, err
	}

	return drv, nil
}

// arrayEach - Call callback method for every array item found in byte slice.
func arrayEach(value []byte, callback func(value []byte, index int) error) error {
	if len(value) < 2 { // Empty array
		return fmt.Errorf("array too short")
	} else if len(value) == 2 {
		return nil
	}

	switch value[0] {
	case '(':
		if value[len(value)-1] != ')' {
			return errArrayNotClosed
		}

	case '[':
		if value[len(value)-1] != ']' {
			return errArrayNotClosed
		}

	default:
		return fmt.Errorf("invalid array opening character: %q", value[0])
	}

	count := 0 // Open paren count
	start := 1 // Start of next value
	idx := 0   // Array index

	escaped := false
	inString := false

	for i, c := range value {
		if escaped { // If value is escaped skip this iteration
			escaped = false

			continue
		} else if c == '\\' { // Set escaped state
			escaped = true

			continue
		}

		if c == '"' {
			inString = !inString

			continue
		} else if inString {
			continue
		}

		if (count == 1 && c == ',') || i == len(value)-1 {
			err := callback(value[start:i], idx)
			if err != nil {
				return err
			}

			idx++ // Array index

			start = i + 1 // Offset to next value
		}

		switch c {
		case '[':
			count++

			continue
		case ']':
			count--

			continue
		case '(':
			count++

			continue
		case ')':
			count--

			continue
		}
	}

	return nil
}

func unquote(b []byte) (string, error) {
	s := string(b)

	unquoted, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("error during unquote of %v: %s", s, err)
	}

	return unquoted, nil
}
