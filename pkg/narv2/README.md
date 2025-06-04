# Fast NAR Reader (fastnar)

This package provides a high-performance NAR (Nix Archive) reader implementation, thanks to the genius brain of @edef. It improves upon the original `pkg/nar` package with a synchronous, state-machine based approach.

### Performance Improvements

1. **Synchronous Processing**: No goroutines or channels overhead
2. **Buffered Reading**: Uses `bufio.Reader` with peek/discard operations
3. **Pre-computed Tokens**: Binary token matching for faster parsing
4. **Lower Memory Allocation**: Minimal object creation during parsing
5. **State Machine**: Direct state transitions without intermediate objects

## Usage Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "io"
    "os"
    
    "github.com/nix-community/go-nix/pkg/fastnar"
)

func main() {
    file, err := os.Open("archive.nar")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    reader := nar.NewReader(file)
    
    for {
        tag, err := reader.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }

        switch tag {
        case nar.TagDir:
            fmt.Printf("Directory: %s\n", reader.Path())
        case nar.TagReg:
            fmt.Printf("File: %s (%d bytes)\n", reader.Path(), reader.Size())
        case nar.TagExe:
            fmt.Printf("Executable: %s (%d bytes)\n", reader.Path(), reader.Size())
        case nar.TagSym:
            fmt.Printf("Symlink: %s -> %s\n", reader.Path(), reader.Target())
        }
    }
}
```

### Copying NAR Archives

```go
// High-performance NAR copying
func copyNAR(dst io.Writer, src io.Reader) error {
    reader := nar.NewReader(src)
    writer := nar.NewWriter(dst)
    return nar.Copy(writer, reader)
}
```

### Reading File Contents

```go
for {
    tag, err := reader.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    if tag == nar.TagReg || tag == nar.TagExe {
        // Read file content
        content := make([]byte, reader.Size())
        _, err := io.ReadFull(reader, content)
        if err != nil {
            return err
        }
        
        fmt.Printf("File %s: %s\n", reader.Path(), string(content))
    }
}
```

## Migration Guide

### From pkg/nar to pkg/fastnar

**Before:**
```go
import "github.com/nix-community/go-nix/pkg/nar"

reader, err := nar.NewReader(file)
if err != nil {
    return err
}
defer reader.Close()

for {
    header, err := reader.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    switch header.Type {
    case nar.TypeDirectory:
        fmt.Printf("Dir: %s\n", header.Path)
    case nar.TypeRegular:
        fmt.Printf("File: %s\n", header.Path)
        if header.Executable {
            fmt.Printf("  (executable)\n")
        }
    case nar.TypeSymlink:
        fmt.Printf("Link: %s -> %s\n", header.Path, header.LinkTarget)
    }
}
```

**After:**
```go
import "github.com/nix-community/go-nix/pkg/fastnar"

reader := nar.NewReader(file)

for {
    tag, err := reader.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    
    switch tag {
    case nar.TagDir:
        fmt.Printf("Dir: %s\n", reader.Path())
    case nar.TagReg:
        fmt.Printf("File: %s\n", reader.Path())
    case nar.TagExe:
        fmt.Printf("File: %s\n", reader.Path())
        fmt.Printf("  (executable)\n")
    case nar.TagSym:
        fmt.Printf("Link: %s -> %s\n", reader.Path(), reader.Target())
    }
}
```
