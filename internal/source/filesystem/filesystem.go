package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"context-os/domain/contracts"
	"context-os/domain/events"
	"context-os/internal/source"
)

const (
	metadataPath                = "path"
	metadataFormat              = "filesystem_format"
	metadataExtension           = "filesystem_extension"
	metadataContentHash         = "filesystem_content_hash"
	metadataModifiedAt          = "filesystem_modified_at"
	metadataSize                = "filesystem_size"
	metadataInclude             = "filesystem_include"
	metadataExclude             = "filesystem_exclude"
	metadataMode                = "filesystem_ingest_mode"
	metadataRoot                = "filesystem_root"
	metadataRelativePath        = "filesystem_relative_path"
	metadataFolderFileCount     = "filesystem_folder_file_count"
	metadataFolderSkippedCount  = "filesystem_folder_skipped_count"
	metadataFolderFirstError    = "filesystem_folder_first_error"
	metadataMaxFiles            = "filesystem_max_files"
	metadataMaxFileSize         = "filesystem_max_file_size"
	metadataSpreadsheetSheets   = "filesystem_spreadsheet_sheets"
	metadataSpreadsheetCells    = "filesystem_spreadsheet_cells"
	metadataSpreadsheetRows     = "filesystem_spreadsheet_rows"
	metadataSpreadsheetFormulas = "filesystem_spreadsheet_formulas"
	defaultFolderMaxFiles       = 1000
	defaultFolderMaxFileSize    = int64(10 << 20)
)

type connector struct {
	base source.MCPConnector
}

type fileExtraction struct {
	Content      string
	Format       string
	Extension    string
	Hash         string
	Size         int64
	ModifiedAt   string
	Spreadsheet  *workbookExtraction
	OpenAPI      *specExtraction
	OriginalPath string
}

type folderFile struct {
	Path         string
	RelativePath string
	Extraction   fileExtraction
}

type folderScan struct {
	Files        []folderFile
	Skipped      int
	FirstSkipped string
}

var errFolderFileLimitReached = errors.New("filesystem folder file limit reached")

// NewConnector returns a filesystem source connector that ingests local file events.
func NewConnector() contracts.MCPSourceConnector {
	return connector{base: source.NewMCPConnector("filesystem", contracts.CapabilityFiles)}
}

// Name returns the connector name for provenance and routing.
func (c connector) Name() string { return c.base.Name() }

// Capabilities returns the connector capabilities supported by this adapter.
func (c connector) Capabilities() []contracts.Capability { return c.base.Capabilities() }

// Ingest reads local files and emits replay-safe raw ingestion events with path provenance.
func (c connector) Ingest(ctx context.Context, req contracts.SourceRequest) ([]events.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
	}

	req.Metadata = cloneMetadata(req.Metadata)
	pathValue := pathFromURI(req.URI)

	if strings.TrimSpace(req.Content) == "" {
		if pathValue == "" {
			return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, errors.New("filesystem uri is required when content is empty"))
		}
		info, err := os.Stat(pathValue)
		if err != nil {
			return nil, c.connectorError(withPathIdentity(req, pathValue, "file"), contracts.ErrorKindInvalidRequest, false, fmt.Errorf("stat filesystem path %q: %w", pathValue, err))
		}
		if info.IsDir() {
			return c.ingestDirectory(ctx, req, pathValue)
		}
		return c.ingestFile(ctx, req, pathValue)
	}

	return c.ingestInline(ctx, req, pathValue)
}

func (c connector) ingestFile(ctx context.Context, req contracts.SourceRequest, pathValue string) ([]events.Event, error) {
	req = withPathIdentity(req, pathValue, "file")
	if err := validatePathRules(pathValue, req.Metadata); err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindPermanent, false, err)
	}
	extracted, err := readAndExtract(pathValue)
	if err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}
	req.Content = extracted.Content
	return c.ingestExtracted(ctx, req, extracted)
}

func (c connector) ingestInline(ctx context.Context, req contracts.SourceRequest, pathValue string) ([]events.Event, error) {
	extracted, err := extractBytes(pathValue, []byte(req.Content))
	if err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}
	if pathValue != "" {
		extracted.OriginalPath = pathValue
	}
	return c.ingestExtracted(ctx, req, extracted)
}

func (c connector) ingestExtracted(ctx context.Context, req contracts.SourceRequest, extracted fileExtraction) ([]events.Event, error) {
	if req.Cursor == "" {
		req.Cursor = extracted.Hash
	}
	enrichMetadata(req.URI, extracted, req.Metadata)

	return c.base.Ingest(ctx, req)
}

func (c connector) ingestDirectory(ctx context.Context, req contracts.SourceRequest, root string) ([]events.Event, error) {
	req = withPathIdentity(req, root, "folder")
	if excluded, pattern := matchesAnyPathPattern(root, req.Metadata[metadataExclude]); excluded {
		err := fmt.Errorf("filesystem directory %q excluded by pattern %q", root, pattern)
		return nil, c.connectorError(req, contracts.ErrorKindPermanent, false, err)
	}

	maxFiles, err := positiveMetadataInt(req.Metadata, metadataMaxFiles, defaultFolderMaxFiles)
	if err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}
	maxFileSize, err := positiveMetadataInt64(req.Metadata, metadataMaxFileSize, defaultFolderMaxFileSize)
	if err != nil {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}

	scan, err := scanDirectory(ctx, root, req.Metadata, maxFiles, maxFileSize)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, c.connectorError(req, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
		}
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, err)
	}
	if len(scan.Files) == 0 {
		return nil, c.connectorError(req, contracts.ErrorKindInvalidRequest, false, emptyFolderError(root, scan))
	}

	out := make([]events.Event, 0, len(scan.Files))
	for _, file := range scan.Files {
		if err := ctx.Err(); err != nil {
			return nil, c.connectorError(req, contracts.ErrorKindCanceled, errors.Is(err, context.DeadlineExceeded), err)
		}
		childReq := contracts.SourceRequest{
			URI:      file.Path,
			Content:  file.Extraction.Content,
			Cursor:   req.Cursor,
			Metadata: folderFileMetadata(req.Metadata, root, file, len(scan.Files), scan.Skipped, scan.FirstSkipped),
		}
		if childReq.Cursor == "" {
			childReq.Cursor = file.Extraction.Hash
		}
		enrichMetadata(childReq.URI, file.Extraction, childReq.Metadata)
		ingested, err := c.base.Ingest(ctx, childReq)
		if err != nil {
			return nil, err
		}
		out = append(out, ingested...)
	}

	return out, nil
}

func scanDirectory(ctx context.Context, root string, metadata map[string]string, maxFiles int, maxFileSize int64) (folderScan, error) {
	scan := folderScan{Files: make([]folderFile, 0)}
	err := filepath.WalkDir(root, func(pathValue string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if pathValue == root {
				return walkErr
			}
			scan.addSkipped(pathValue, walkErr.Error())
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if pathValue == root {
			return nil
		}
		if entry.IsDir() {
			if excluded, pattern := matchesAnyPathPattern(pathValue, metadata[metadataExclude]); excluded {
				scan.addSkipped(pathValue, fmt.Sprintf("excluded by pattern %q", pattern))
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			scan.addSkipped(pathValue, "symbolic links are skipped in folder ingestion")
			return nil
		}
		if err := validatePathRules(pathValue, metadata); err != nil {
			scan.addSkipped(pathValue, err.Error())
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			scan.addSkipped(pathValue, err.Error())
			return nil
		}
		if info.Size() > maxFileSize {
			scan.addSkipped(pathValue, fmt.Sprintf("file size %d exceeds limit %d", info.Size(), maxFileSize))
			return nil
		}
		if len(scan.Files) >= maxFiles {
			scan.addSkipped(pathValue, fmt.Sprintf("file count limit %d reached", maxFiles))
			return errFolderFileLimitReached
		}
		extracted, err := readAndExtract(pathValue)
		if err != nil {
			scan.addSkipped(pathValue, err.Error())
			return nil
		}
		relativePath, err := filepath.Rel(root, pathValue)
		if err != nil {
			scan.addSkipped(pathValue, err.Error())
			return nil
		}
		scan.Files = append(scan.Files, folderFile{
			Path:         pathValue,
			RelativePath: filepath.ToSlash(relativePath),
			Extraction:   extracted,
		})
		return nil
	})
	if errors.Is(err, errFolderFileLimitReached) {
		return scan, nil
	}
	return scan, err
}

func (s *folderScan) addSkipped(pathValue, reason string) {
	s.Skipped++
	if s.FirstSkipped != "" {
		return
	}
	s.FirstSkipped = pathValue + ": " + reason
}

func folderFileMetadata(base map[string]string, root string, file folderFile, fileCount, skippedCount int, firstSkipped string) map[string]string {
	metadata := cloneMetadata(base)
	metadata[contracts.MetadataObjectType] = "file"
	delete(metadata, contracts.MetadataObjectID)
	delete(metadata, events.MetadataEventID)
	delete(metadata, events.MetadataSourceID)
	metadata[metadataMode] = "folder"
	metadata[metadataRoot] = root
	metadata[metadataRelativePath] = file.RelativePath
	metadata[metadataFolderFileCount] = strconv.Itoa(fileCount)
	metadata[metadataFolderSkippedCount] = strconv.Itoa(skippedCount)
	if firstSkipped != "" {
		metadata[metadataFolderFirstError] = firstSkipped
	}
	return metadata
}

func emptyFolderError(root string, scan folderScan) error {
	if scan.FirstSkipped == "" {
		return fmt.Errorf("filesystem directory %q produced no ingestible files", root)
	}
	return fmt.Errorf("filesystem directory %q produced no ingestible files; first skipped path: %s", root, scan.FirstSkipped)
}

func positiveMetadataInt(metadata map[string]string, key string, fallback int) (int, error) {
	value := strings.TrimSpace(metadata[key])
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("metadata %s must be a positive integer", key)
	}
	return parsed, nil
}

func positiveMetadataInt64(metadata map[string]string, key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(metadata[key])
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("metadata %s must be a positive integer", key)
	}
	return parsed, nil
}

func withPathIdentity(req contracts.SourceRequest, pathValue, objectType string) contracts.SourceRequest {
	if req.Metadata == nil {
		req.Metadata = map[string]string{}
	}
	setIfMissing(req.Metadata, contracts.MetadataObjectType, objectType)
	setIfMissing(req.Metadata, contracts.MetadataObjectID, pathValue)
	setIfMissing(req.Metadata, events.MetadataSourceID, "filesystem:"+objectType+":"+pathValue)
	return req
}

func readAndExtract(pathValue string) (fileExtraction, error) {
	info, err := os.Stat(pathValue)
	if err != nil {
		return fileExtraction{}, fmt.Errorf("stat filesystem path %q: %w", pathValue, err)
	}
	if info.IsDir() {
		return fileExtraction{}, fmt.Errorf("filesystem path %q is a directory; provide a file path", pathValue)
	}

	data, err := os.ReadFile(pathValue)
	if err != nil {
		return fileExtraction{}, fmt.Errorf("read filesystem path %q: %w", pathValue, err)
	}

	extracted, err := extractBytes(pathValue, data)
	if err != nil {
		return fileExtraction{}, err
	}
	extracted.Size = info.Size()
	extracted.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339Nano)
	extracted.OriginalPath = pathValue
	return extracted, nil
}

func extractBytes(pathValue string, data []byte) (fileExtraction, error) {
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(pathValue)), ".")
	extracted := fileExtraction{
		Extension:    extension,
		Format:       formatForExtension(extension),
		Hash:         hashBytes(data),
		Size:         int64(len(data)),
		OriginalPath: pathValue,
	}

	switch extension {
	case "csv", "xlsx":
		workbook, err := extractWorkbook(filepath.Base(pathValue), data)
		if err != nil {
			return fileExtraction{}, err
		}
		extracted.Content = workbook.Content
		extracted.Spreadsheet = &workbook
	case "docx":
		content, err := extractWordText(data)
		if err != nil {
			return fileExtraction{}, err
		}
		extracted.Content = content
	case "pptx":
		content, err := extractPowerPointText(data)
		if err != nil {
			return fileExtraction{}, err
		}
		extracted.Content = content
	case "pdf":
		extracted.Content = extractPDFText(data)
	case "json", "yaml", "yml":
		extracted.Content = string(data)
		if spec, ok := extractOpenAPISpec(data); ok {
			extracted.Format = "openapi_spec"
			extracted.OpenAPI = &spec
		}
	default:
		if content, ok := extractTextContent(extension, data); ok {
			extracted.Content = content
			return extracted, nil
		}
		return fileExtraction{}, fmt.Errorf("unsupported binary filesystem format %q", extension)
	}

	return extracted, nil
}

func pathFromURI(uri string) string {
	trimmed := strings.TrimSpace(uri)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" {
		return trimmed
	}
	if parsed.Scheme == "file" {
		path, decodeErr := url.PathUnescape(parsed.Path)
		if decodeErr != nil {
			return parsed.Path
		}
		return path
	}
	if parsed.Scheme == "filesystem" {
		path := parsed.Host + parsed.Path
		decoded, decodeErr := url.PathUnescape(path)
		if decodeErr != nil {
			return path
		}
		return decoded
	}
	return ""
}

func validatePathRules(pathValue string, metadata map[string]string) error {
	if excluded, pattern := matchesAnyPathPattern(pathValue, metadata[metadataExclude]); excluded {
		return fmt.Errorf("filesystem path %q excluded by pattern %q", pathValue, pattern)
	}
	include := strings.TrimSpace(metadata[metadataInclude])
	if include == "" {
		return nil
	}
	if included, _ := matchesAnyPathPattern(pathValue, include); included {
		return nil
	}
	return fmt.Errorf("filesystem path %q does not match include rules", pathValue)
}

func matchesAnyPathPattern(pathValue, patternList string) (bool, string) {
	pathValue = filepath.ToSlash(pathValue)
	base := filepath.Base(pathValue)
	for _, pattern := range strings.Split(patternList, ",") {
		pattern = strings.TrimSpace(filepath.ToSlash(pattern))
		if pattern == "" {
			continue
		}
		matched, err := filepath.Match(pattern, pathValue)
		if err == nil && matched {
			return true, pattern
		}
		matched, err = filepath.Match(pattern, base)
		if err == nil && matched {
			return true, pattern
		}
		if strings.Contains(pattern, "**") && globstarMatch(pathValue, pattern) {
			return true, pattern
		}
	}
	return false, ""
}

func globstarMatch(pathValue, pattern string) bool {
	parts := strings.Split(pattern, "**")
	position := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		index := strings.Index(pathValue[position:], part)
		if index < 0 {
			return false
		}
		position += index + len(part)
	}
	return true
}

func enrichMetadata(uri string, extracted fileExtraction, metadata map[string]string) {
	objectID := stableObjectID(uri, extracted)
	setIfMissing(metadata, contracts.MetadataObjectType, "file")
	setIfMissing(metadata, contracts.MetadataObjectID, objectID)
	setIfMissing(metadata, events.MetadataSourceID, "filesystem:file:"+objectID)
	setIfMissing(metadata, metadataPath, extracted.OriginalPath)
	setIfMissing(metadata, metadataFormat, extracted.Format)
	setIfMissing(metadata, metadataExtension, extracted.Extension)
	setIfMissing(metadata, metadataContentHash, extracted.Hash)
	setIfMissing(metadata, metadataModifiedAt, extracted.ModifiedAt)
	setIfMissing(metadata, metadataSize, strconv.FormatInt(extracted.Size, 10))
	if extracted.Spreadsheet != nil {
		setIfMissing(metadata, metadataSpreadsheetSheets, strings.Join(extracted.Spreadsheet.Sheets, ","))
		setIfMissing(metadata, metadataSpreadsheetCells, strconv.Itoa(extracted.Spreadsheet.CellCount))
		setIfMissing(metadata, metadataSpreadsheetRows, strconv.Itoa(extracted.Spreadsheet.RowCount))
		setIfMissing(metadata, metadataSpreadsheetFormulas, strconv.Itoa(extracted.Spreadsheet.FormulaCount))
	}
	if extracted.OpenAPI != nil {
		enrichOpenAPIMetadata(*extracted.OpenAPI, metadata)
	}
}

func stableObjectID(uri string, extracted fileExtraction) string {
	if pathValue := strings.TrimSpace(extracted.OriginalPath); pathValue != "" {
		return pathValue
	}
	if trimmed := strings.TrimSpace(uri); trimmed != "" {
		return trimmed
	}
	return extracted.Hash
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (c connector) connectorError(req contracts.SourceRequest, kind contracts.ErrorKind, retryable bool, err error) error {
	return &contracts.ConnectorError{
		Connector:  c.base.Name(),
		URI:        req.URI,
		ObjectType: req.Metadata[contracts.MetadataObjectType],
		ObjectID:   req.Metadata[contracts.MetadataObjectID],
		Kind:       kind,
		Retryable:  retryable,
		Err:        err,
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func setIfMissing(metadata map[string]string, key, value string) {
	if value == "" {
		return
	}
	if _, exists := metadata[key]; exists {
		return
	}
	metadata[key] = value
}
