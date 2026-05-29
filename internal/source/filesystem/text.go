package filesystem

import "unicode/utf8"

func extractTextContent(extension string, data []byte) (string, bool) {
	if isTextExtension(extension) || utf8.Valid(data) {
		return string(data), true
	}
	return "", false
}

func formatForExtension(extension string) string {
	switch extension {
	case "csv", "xlsx":
		return "spreadsheet"
	case "docx":
		return "word_document"
	case "pptx":
		return "presentation"
	case "pdf":
		return "pdf"
	default:
		if extension == "" {
			return "text"
		}
		if isTextExtension(extension) {
			return "text"
		}
		return extension
	}
}

func isTextExtension(extension string) bool {
	switch extension {
	case "", "txt", "md", "markdown", "go", "yaml", "yml", "json", "ts", "tsx", "js", "jsx", "py", "rb", "java", "kt", "rs", "c", "h", "cpp", "hpp", "cs", "sql", "toml", "xml", "html", "css", "scss", "sh", "bash", "zsh", "env", "ini", "cfg", "conf":
		return true
	default:
		return false
	}
}
