package googledrive

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
)

// formatCSVAsTable converts a Google Sheets CSV export into tab-separated text while preserving rows.
func formatCSVAsTable(body []byte) (string, error) {
	reader := csv.NewReader(strings.NewReader(string(body)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("parse csv export: %w", err)
	}

	formatted := make([]string, 0, len(rows))
	for _, row := range rows {
		formatted = append(formatted, strings.Join(row, "\t"))
	}
	return strings.Join(formatted, "\n"), nil
}

// formatSlidesAsText extracts visible text runs from Google Slides API JSON and groups them by slide.
func formatSlidesAsText(body []byte) (string, error) {
	var presentation struct {
		Slides []struct {
			PageElements []struct {
				Shape *struct {
					Text struct {
						TextElements []struct {
							TextRun *struct {
								Content string `json:"content"`
							} `json:"textRun"`
						} `json:"textElements"`
					} `json:"text"`
				} `json:"shape"`
			} `json:"pageElements"`
		} `json:"slides"`
	}
	if err := json.Unmarshal(body, &presentation); err != nil {
		return "", fmt.Errorf("decode slides presentation: %w", err)
	}

	sections := make([]string, 0, len(presentation.Slides))
	for index, slide := range presentation.Slides {
		lines := make([]string, 0)
		for _, element := range slide.PageElements {
			if element.Shape == nil {
				continue
			}
			for _, textElement := range element.Shape.Text.TextElements {
				if textElement.TextRun == nil {
					continue
				}
				content := strings.TrimSpace(textElement.TextRun.Content)
				if content == "" {
					continue
				}
				lines = append(lines, content)
			}
		}
		if len(lines) == 0 {
			continue
		}
		sections = append(sections, fmt.Sprintf("Slide %d\n%s", index+1, strings.Join(lines, "\n")))
	}
	return strings.Join(sections, "\n\n"), nil
}
