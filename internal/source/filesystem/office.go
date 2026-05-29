package filesystem

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func extractWordText(data []byte) (string, error) {
	return extractOfficeText(data, func(name string) bool {
		return name == "word/document.xml" || strings.HasPrefix(name, "word/header") || strings.HasPrefix(name, "word/footer")
	})
}

func extractPowerPointText(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pptx: %w", err)
	}

	var names []string
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var lines []string
	for index, name := range names {
		file, ok := openZipFile(reader, name)
		if !ok {
			continue
		}
		text, err := extractParagraphText(file)
		_ = file.Close()
		if err != nil {
			return "", err
		}
		for _, line := range splitNonEmptyLines(text) {
			lines = append(lines, fmt.Sprintf("Slide %d: %s", index+1, line))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func extractOfficeText(data []byte, include func(string) bool) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open office document: %w", err)
	}

	var names []string
	for _, file := range reader.File {
		name := filepath.ToSlash(file.Name)
		if include(name) && strings.HasSuffix(name, ".xml") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	var parts []string
	for _, name := range names {
		file, ok := openZipFile(reader, name)
		if !ok {
			continue
		}
		text, err := extractParagraphText(file)
		_ = file.Close()
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func extractParagraphText(reader io.Reader) (string, error) {
	decoder := xml.NewDecoder(reader)
	var lines []string
	var paragraph strings.Builder
	inText := false
	paragraphDepth := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("parse office xml: %w", err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Local == "p" {
				paragraphDepth++
			}
			if typed.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inText {
				paragraph.Write([]byte(typed))
			}
		case xml.EndElement:
			if typed.Name.Local == "t" {
				inText = false
			}
			if typed.Name.Local == "p" && paragraphDepth > 0 {
				paragraphDepth--
				line := strings.TrimSpace(paragraph.String())
				if line != "" {
					lines = append(lines, line)
				}
				paragraph.Reset()
			}
		}
	}

	if line := strings.TrimSpace(paragraph.String()); line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

func splitNonEmptyLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
