package filesystem

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type workbookExtraction struct {
	Content      string
	Workbook     string
	Format       string
	Hash         string
	Sheets       []string
	RowCount     int
	CellCount    int
	FormulaCount int
}

type workbookSheet struct {
	Name   string
	Target string
}

type worksheetExtraction struct {
	Lines        []string
	Rows         int
	Cells        int
	FormulaCount int
}

func extractWorkbook(name string, data []byte) (workbookExtraction, error) {
	workbook := strings.TrimSpace(name)
	if workbook == "" {
		workbook = "workbook"
	}

	extraction := workbookExtraction{
		Workbook: workbook,
		Format:   workbookFormat(workbook),
		Hash:     hashBytes(data),
	}

	switch extraction.Format {
	case "csv":
		return extractCSV(extraction, data)
	case "xlsx":
		return extractXLSX(extraction, data)
	default:
		extraction.Content = string(data)
		return extraction, nil
	}
}

func extractCSV(extraction workbookExtraction, data []byte) (workbookExtraction, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return workbookExtraction{}, fmt.Errorf("parse csv workbook: %w", err)
	}

	sheetName := "Sheet1"
	lines := make([]string, 0)
	for rowIndex, record := range records {
		for colIndex, value := range record {
			cellRef := cellName(colIndex+1, rowIndex+1)
			lines = append(lines, fmt.Sprintf("%s!%s=%s", sheetName, cellRef, value))
			extraction.CellCount++
			if strings.HasPrefix(strings.TrimSpace(value), "=") {
				extraction.FormulaCount++
				lines = append(lines, fmt.Sprintf("%s!%s.formula=%s", sheetName, cellRef, strings.TrimPrefix(strings.TrimSpace(value), "=")))
			}
		}
	}

	extraction.Sheets = []string{sheetName}
	extraction.RowCount = len(records)
	extraction.Content = strings.Join(lines, "\n")
	return extraction, nil
}

func extractXLSX(extraction workbookExtraction, data []byte) (workbookExtraction, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return workbookExtraction{}, fmt.Errorf("open xlsx workbook: %w", err)
	}

	sharedStrings, err := readSharedStrings(reader)
	if err != nil {
		return workbookExtraction{}, err
	}

	sheets, err := readWorkbookSheets(reader)
	if err != nil {
		return workbookExtraction{}, err
	}
	if len(sheets) == 0 {
		sheets = fallbackWorksheetList(reader)
	}

	var lines []string
	for _, sheet := range sheets {
		worksheet, ok := openZipFile(reader, sheet.Target)
		if !ok {
			continue
		}

		extractedSheet, err := extractWorksheet(sheet.Name, worksheet, sharedStrings)
		_ = worksheet.Close()
		if err != nil {
			return workbookExtraction{}, err
		}
		lines = append(lines, extractedSheet.Lines...)
		extraction.Sheets = append(extraction.Sheets, sheet.Name)
		extraction.RowCount += extractedSheet.Rows
		extraction.CellCount += extractedSheet.Cells
		extraction.FormulaCount += extractedSheet.FormulaCount
	}

	extraction.Content = strings.Join(lines, "\n")
	return extraction, nil
}

func readSharedStrings(reader *zip.Reader) ([]string, error) {
	file, ok := openZipFile(reader, "xl/sharedStrings.xml")
	if !ok {
		return nil, nil
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	var values []string
	var current strings.Builder
	inText := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse xlsx shared strings: %w", err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			if typed.Name.Local == "si" {
				current.Reset()
			}
			if typed.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inText {
				current.Write([]byte(typed))
			}
		case xml.EndElement:
			if typed.Name.Local == "t" {
				inText = false
			}
			if typed.Name.Local == "si" {
				values = append(values, current.String())
			}
		}
	}

	return values, nil
}

func readWorkbookSheets(reader *zip.Reader) ([]workbookSheet, error) {
	rels, err := readWorkbookRelationships(reader)
	if err != nil {
		return nil, err
	}

	file, ok := openZipFile(reader, "xl/workbook.xml")
	if !ok {
		return nil, nil
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	var sheets []workbookSheet
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse xlsx workbook: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sheet" {
			continue
		}

		var name string
		var relID string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "name":
				name = attr.Value
			case "id":
				relID = attr.Value
			}
		}
		if name == "" {
			name = fmt.Sprintf("Sheet%d", len(sheets)+1)
		}
		target := rels[relID]
		if target == "" {
			target = fmt.Sprintf("xl/worksheets/sheet%d.xml", len(sheets)+1)
		}
		sheets = append(sheets, workbookSheet{Name: name, Target: normalizeWorkbookTarget(target)})
	}

	return sheets, nil
}

func readWorkbookRelationships(reader *zip.Reader) (map[string]string, error) {
	file, ok := openZipFile(reader, "xl/_rels/workbook.xml.rels")
	if !ok {
		return map[string]string{}, nil
	}
	defer func() { _ = file.Close() }()

	decoder := xml.NewDecoder(file)
	rels := map[string]string{}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parse xlsx workbook relationships: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Relationship" {
			continue
		}

		var id string
		var target string
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "Id":
				id = attr.Value
			case "Target":
				target = attr.Value
			}
		}
		if id != "" && target != "" {
			rels[id] = target
		}
	}

	return rels, nil
}

func fallbackWorksheetList(reader *zip.Reader) []workbookSheet {
	var names []string
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/") && strings.HasSuffix(file.Name, ".xml") {
			names = append(names, file.Name)
		}
	}
	sort.Strings(names)

	sheets := make([]workbookSheet, 0, len(names))
	for index, name := range names {
		sheets = append(sheets, workbookSheet{Name: fmt.Sprintf("Sheet%d", index+1), Target: name})
	}
	return sheets
}

func extractWorksheet(sheetName string, worksheet io.Reader, sharedStrings []string) (worksheetExtraction, error) {
	decoder := xml.NewDecoder(worksheet)
	var out worksheetExtraction
	seenRows := map[int]struct{}{}

	var cellRef string
	var cellType string
	var value strings.Builder
	var inline strings.Builder
	var formula strings.Builder
	inValue := false
	inInline := false
	inFormula := false
	inCell := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return worksheetExtraction{}, fmt.Errorf("parse xlsx worksheet %q: %w", sheetName, err)
		}

		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "c":
				inCell = true
				cellRef = ""
				cellType = ""
				value.Reset()
				inline.Reset()
				formula.Reset()
				for _, attr := range typed.Attr {
					switch attr.Name.Local {
					case "r":
						cellRef = attr.Value
					case "t":
						cellType = attr.Value
					}
				}
			case "v":
				if inCell {
					inValue = true
				}
			case "t":
				if inCell {
					inInline = true
				}
			case "f":
				if inCell {
					inFormula = true
				}
			}
		case xml.CharData:
			if inFormula {
				formula.Write([]byte(typed))
				continue
			}
			if inValue {
				value.Write([]byte(typed))
				continue
			}
			if inInline {
				inline.Write([]byte(typed))
			}
		case xml.EndElement:
			switch typed.Name.Local {
			case "v":
				inValue = false
			case "t":
				inInline = false
			case "f":
				inFormula = false
			case "c":
				text := resolveCellValue(cellType, value.String(), inline.String(), sharedStrings)
				formulaText := strings.TrimSpace(formula.String())
				if text != "" || formulaText != "" {
					if cellRef == "" {
						cellRef = cellName(out.Cells+1, 1)
					}
					out.Lines = append(out.Lines, fmt.Sprintf("%s!%s=%s", sheetName, cellRef, text))
					out.Cells++
					if row := rowFromCell(cellRef); row > 0 {
						seenRows[row] = struct{}{}
					}
				}
				if formulaText != "" {
					out.Lines = append(out.Lines, fmt.Sprintf("%s!%s.formula=%s", sheetName, cellRef, formulaText))
					out.FormulaCount++
				}
				inCell = false
			}
		}
	}

	out.Rows = len(seenRows)
	return out, nil
}

func resolveCellValue(cellType, rawValue, inlineValue string, sharedStrings []string) string {
	if cellType == "inlineStr" {
		return strings.TrimSpace(inlineValue)
	}
	if cellType == "s" {
		index, err := strconv.Atoi(strings.TrimSpace(rawValue))
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return strings.TrimSpace(rawValue)
		}
		return strings.TrimSpace(sharedStrings[index])
	}
	return strings.TrimSpace(rawValue)
}

func normalizeWorkbookTarget(target string) string {
	target = filepath.ToSlash(strings.TrimLeft(target, "/"))
	if strings.HasPrefix(target, "xl/") {
		return target
	}
	return "xl/" + target
}

func cellName(column, row int) string {
	return columnName(column) + strconv.Itoa(row)
}

func columnName(column int) string {
	if column <= 0 {
		return "A"
	}

	var chars []byte
	for column > 0 {
		column--
		chars = append([]byte{byte('A' + column%26)}, chars...)
		column /= 26
	}
	return string(chars)
}

func rowFromCell(cell string) int {
	start := -1
	for index, r := range cell {
		if r >= '0' && r <= '9' {
			start = index
			break
		}
	}
	if start < 0 {
		return 0
	}
	row, err := strconv.Atoi(cell[start:])
	if err != nil {
		return 0
	}
	return row
}

func workbookFormat(name string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	if ext == "" {
		return "text"
	}
	return ext
}
