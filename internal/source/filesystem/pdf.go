package filesystem

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
)

func extractPDFText(data []byte) string {
	blobs := [][]byte{data}
	blobs = append(blobs, inflatedPDFStreams(data)...)

	seen := map[string]struct{}{}
	var lines []string
	for _, blob := range blobs {
		for _, text := range pdfLiteralStrings(blob) {
			line := normalizeWhitespace(text)
			if line == "" || !hasLetterOrDigit(line) {
				continue
			}
			if _, exists := seen[line]; exists {
				continue
			}
			seen[line] = struct{}{}
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func inflatedPDFStreams(data []byte) [][]byte {
	var streams [][]byte
	remaining := data
	for {
		start := bytes.Index(remaining, []byte("stream"))
		if start < 0 {
			break
		}
		streamStart := start + len("stream")
		if streamStart < len(remaining) && remaining[streamStart] == '\r' {
			streamStart++
		}
		if streamStart < len(remaining) && remaining[streamStart] == '\n' {
			streamStart++
		}
		end := bytes.Index(remaining[streamStart:], []byte("endstream"))
		if end < 0 {
			break
		}
		streamData := bytes.TrimSpace(remaining[streamStart : streamStart+end])
		reader, err := zlib.NewReader(bytes.NewReader(streamData))
		if err == nil {
			inflated, readErr := io.ReadAll(reader)
			_ = reader.Close()
			if readErr == nil {
				streams = append(streams, inflated)
			}
		} else {
			streams = append(streams, streamData)
		}
		remaining = remaining[streamStart+end+len("endstream"):]
	}
	return streams
}

func pdfLiteralStrings(data []byte) []string {
	var values []string
	for index := 0; index < len(data); index++ {
		if data[index] != '(' {
			continue
		}
		text, next := readPDFLiteral(data, index+1)
		if next <= index {
			continue
		}
		values = append(values, text)
		index = next
	}
	return values
}

func readPDFLiteral(data []byte, start int) (string, int) {
	var out strings.Builder
	depth := 1
	escaped := false
	for index := start; index < len(data); index++ {
		current := data[index]
		if escaped {
			switch current {
			case 'n':
				out.WriteByte('\n')
			case 'r':
				out.WriteByte('\r')
			case 't':
				out.WriteByte('\t')
			case 'b':
				out.WriteByte('\b')
			case 'f':
				out.WriteByte('\f')
			default:
				out.WriteByte(current)
			}
			escaped = false
			continue
		}

		switch current {
		case '\\':
			escaped = true
		case '(':
			depth++
			out.WriteByte(current)
		case ')':
			depth--
			if depth == 0 {
				return out.String(), index
			}
			out.WriteByte(current)
		default:
			out.WriteByte(current)
		}
	}
	return "", start
}

func normalizeWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func hasLetterOrDigit(text string) bool {
	for _, r := range text {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return true
		}
	}
	return false
}
