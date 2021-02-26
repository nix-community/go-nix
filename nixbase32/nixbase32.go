package nixbase32

import "encoding/base32"

var NixEncoding = base32.NewEncoding("0123456789abcdfghijklmnpqrsvwxyz")
