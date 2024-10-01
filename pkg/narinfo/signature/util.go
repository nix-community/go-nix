package signature

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// decode parses a "<name>:<base64-data>" string into a name, and data pair.
// And then checks that the data is of dataSize.
//
// This is used internally for all the data structures below in this file.
func decode(s string, dataSize int) (name string, data []byte, err error) {
	kv := strings.SplitN(s, ":", 2)
	name = kv[0]

	var dataStr string

	if len(kv) != 2 {
		return "", nil, fmt.Errorf("encountered invalid number of fields: %v", len(kv))
	}

	dataStr = kv[1]

	if name == "" {
		return "", nil, fmt.Errorf("name is missing")
	}

	if dataStr == "" {
		return "", nil, fmt.Errorf("data is missing")
	}

	data, err = base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return "", nil, fmt.Errorf("data is corrupt: %w", err)
	}

	if len(data) != dataSize {
		return "", nil, fmt.Errorf("data is not the right size: expected %d but got %d", dataSize, len(data))
	}

	return name, data, nil
}

// encode is the counterpart of the decode function above. Generate a
// "<name>:<base64-data>" string from the underlying data structures.
func encode(name string, data []byte) string {
	return name + ":" + base64.StdEncoding.EncodeToString(data)
}
