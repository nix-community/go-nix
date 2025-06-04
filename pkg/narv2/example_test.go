package narv2_test

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/nix-community/go-nix/pkg/narv2"
	oldnar "github.com/nix-community/go-nix/pkg/nar"
)

func ExampleReader() {
	// Create a simple NAR for demonstration
	var buf bytes.Buffer
	w := narv2.NewWriter(&buf)

	// Build a simple directory structure
	w.Directory()
	
	w.Entry("file.txt")
	w.File(false, 5)
	w.Write([]byte("hello"))
	w.Close()

	w.Entry("link")
	w.Link("file.txt")

	w.Entry("script.sh")
	w.File(true, 11)
	w.Write([]byte("#!/bin/bash"))
	w.Close()

	w.Close() // Close root directory

	narData := buf.Bytes()

	// Read with Reader
	fmt.Println("=== FastReader ===")
	r := narv2.NewReader(bytes.NewReader(narData))
	
	for {
		tag, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		switch tag {
		case narv2.TagDir:
			fmt.Printf("Directory: %s\n", r.Path())
		case narv2.TagReg:
			fmt.Printf("Regular file: %s (size: %d)\n", r.Path(), r.Size())
			// Read file content
			content := make([]byte, r.Size())
			io.ReadFull(r, content)
			fmt.Printf("  Content: %s\n", string(content))
		case narv2.TagExe:
			fmt.Printf("Executable: %s (size: %d)\n", r.Path(), r.Size())
			// Read file content
			content := make([]byte, r.Size())
			io.ReadFull(r, content)
			fmt.Printf("  Content: %s\n", string(content))
		case narv2.TagSym:
			fmt.Printf("Symlink: %s -> %s\n", r.Path(), r.Target())
		}
	}

	// Read with traditional NAR reader for comparison
	fmt.Println("\n=== Traditional Reader ===")
	oldReader, err := oldnar.NewReader(bytes.NewReader(narData))
	if err != nil {
		log.Fatal(err)
	}
	defer oldReader.Close()

	for {
		hdr, err := oldReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		switch hdr.Type {
		case oldnar.TypeDirectory:
			fmt.Printf("Directory: %s\n", hdr.Path)
		case oldnar.TypeRegular:
			if hdr.Executable {
				fmt.Printf("Executable: %s (size: %d)\n", hdr.Path, hdr.Size)
			} else {
				fmt.Printf("Regular file: %s (size: %d)\n", hdr.Path, hdr.Size)
			}
			// Read file content
			if hdr.Size > 0 {
				content := make([]byte, hdr.Size)
				io.ReadFull(oldReader, content)
				fmt.Printf("  Content: %s\n", string(content))
			}
		case oldnar.TypeSymlink:
			fmt.Printf("Symlink: %s -> %s\n", hdr.Path, hdr.LinkTarget)
		}
	}

	// Output:
	// === FastReader ===
	// Directory: /
	// Regular file: /file.txt (size: 5)
	//   Content: hello
	// Symlink: /link -> file.txt
	// Executable: /script.sh (size: 11)
	//   Content: #!/bin/bash
	//
	// === Traditional Reader ===
	// Directory: /
	// Regular file: /file.txt (size: 5)
	//   Content: hello
	// Symlink: /link -> file.txt
	// Executable: /script.sh (size: 11)
	//   Content: #!/bin/bash
}

func ExampleReader_performance() {
	// Performance-focused usage example
	var buf bytes.Buffer
	w := narv2.NewWriter(&buf)

	// Create a larger directory structure
	w.Directory()
	for i := 0; i < 100; i++ {
		w.Entry(fmt.Sprintf("file%d.txt", i))
		w.File(false, 10)
		w.Write([]byte(fmt.Sprintf("content%03d", i)))
		w.Close()
	}
	w.Close()

	narData := buf.Bytes()

	// Reader - synchronous, low overhead
	fmt.Println("=== Reader (synchronous) ===")
	r := narv2.NewReader(bytes.NewReader(narData))
	
	fileCount := 0
	totalSize := uint64(0)
	
	for {
		tag, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		if tag == narv2.TagReg || tag == narv2.TagExe {
			fileCount++
			totalSize += r.Size()
			// Skip reading content for performance
			io.Copy(io.Discard, r)
		}
	}

	fmt.Printf("Files processed: %d\n", fileCount)
	fmt.Printf("Total size: %d bytes\n", totalSize)

	// Output:
	// === Reader (synchronous) ===
	// Files processed: 100
	// Total size: 1000 bytes
}