package narv2

import "io"

func Copy(dst Writer, src Reader) error {
	tag, err := src.Next()
	if err != nil {
		return err
	}
	return copyNAR(dst, src, tag)
}

func copyNAR(dst Writer, src Reader, tag Tag) error {
	switch tag {
	default:
		panic("invalid tag")
	case TagSym:
		return dst.Link(src.Target())
	case TagReg, TagExe:
		if err := dst.File(tag == TagExe, src.Size()); err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		return dst.Close()
	case TagDir:
		if err := dst.Directory(); err != nil {
			return err
		}
		for {
			tag, err := src.Next()
			if err == io.EOF {
				return dst.Close()
			}
			if err != nil {
				return err
			}
			if err := dst.Entry(src.Name()); err != nil {
				return err
			}
			if err := copyNAR(dst, src, tag); err != nil {
				return err
			}
		}
	}
}
