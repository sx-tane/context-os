package filesystem

import (
	"archive/zip"
	"io"
	"path/filepath"
)

func openZipFile(reader *zip.Reader, name string) (io.ReadCloser, bool) {
	normalized := filepath.ToSlash(name)
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != normalized {
			continue
		}
		opened, err := file.Open()
		if err != nil {
			return nil, false
		}
		return opened, true
	}
	return nil, false
}
