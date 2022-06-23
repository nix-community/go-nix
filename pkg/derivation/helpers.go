package derivation

import (
	"reflect"
	"unsafe"
)

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func unsafeGetBytes(s string) []byte {
	return unsafe.Slice(
		(*byte)(
			unsafe.Pointer(
				(*reflect.StringHeader)(unsafe.Pointer(&s)).Data,
			),
		),
		len(s),
	)
}
