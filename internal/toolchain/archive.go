package toolchain

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
)

// extract gives the bytes of the program at a path inside an archive. kit reads .tar.gz and .zip, which Go's own
// library opens, and nothing else.
func extract(name string, body []byte, program string) ([]byte, error) {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		z, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		r := tar.NewReader(z)
		for {
			header, err := r.Next()
			switch {
			case errors.Is(err, io.EOF):
				return nil, fmt.Errorf("%s holds no %s", name, program)
			case err != nil:
				return nil, fmt.Errorf("%s: %w", name, err)
			case strings.TrimPrefix(header.Name, "./") == program:
				return io.ReadAll(r)
			}
		}
	case strings.HasSuffix(name, ".zip"):
		r, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for _, f := range r.File {
			if f.Name == program {
				opened, err := f.Open()
				var program []byte
				if err == nil {
					program, err = io.ReadAll(opened)
				}
				return program, err
			}
		}
		return nil, fmt.Errorf("%s holds no %s", name, program)
	}
	return nil, fmt.Errorf("kit reads .tar.gz and .zip archives alone, and %s is neither", name)
}
