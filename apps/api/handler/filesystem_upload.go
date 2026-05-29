package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"context-os/apps/api/response"
	filesystemsource "context-os/internal/source/filesystem"
)

const (
	filesystemUploadMaxRequestBytes = int64(200 << 20)
	filesystemUploadMemoryBytes     = int64(32 << 20)
	filesystemUploadRootDefault     = "storage/raw/uploads"
	metadataFilesystemUploadID      = "filesystem_upload_id"
	metadataFilesystemUploadRoot    = "filesystem_upload_root"
	metadataFilesystemUploadFiles   = "filesystem_upload_file_count"
	metadataFilesystemUploadName    = "filesystem_upload_original_name"
)

var errInvalidUploadPath = errors.New("invalid upload path")

type stagedFilesystemUpload struct {
	UploadID     string
	Root         string
	IngestURI    string
	FileCount    int
	OriginalName string
}

type uploadFileEntry struct {
	File         *multipart.FileHeader
	RelativePath string
}

// FilesystemUpload handles POST /filesystem/upload by staging browser-uploaded files or folders
// and ingesting them through the filesystem connector.
//
// @Summary      Upload and ingest browser files or folders
// @Description  Accepts multipart/form-data with one or more file parts and optional relative paths, stages them under storage/raw/uploads/<upload-id>/, and ingests through the filesystem connector. Metadata keys filesystem_upload_id, filesystem_upload_root, filesystem_upload_file_count, and filesystem_upload_original_name are added to each result event.
// @Tags         filesystem
// @Accept       multipart/form-data
// @Produce      json
// @Param        files  formData  file    true   "One or more files to upload"
// @Param        paths  formData  string  false  "Relative paths for browser folder uploads (one per file, matching file order)"
// @Success      200    {object}  response.Ingest
// @Failure      400    {object}  map[string]string
// @Failure      405    {object}  map[string]string
// @Failure      500    {object}  map[string]string
// @Failure      502    {object}  map[string]string
// @Failure      503    {object}  map[string]string
// @Router       /filesystem/upload [post]
func FilesystemUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, filesystemUploadMaxRequestBytes)
	if err := r.ParseMultipartForm(filesystemUploadMemoryBytes); err != nil {
		response.WriteError(w, http.StatusBadRequest, "invalid_multipart", err.Error())
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	files := uploadedFileEntries(r.MultipartForm)
	if len(files) == 0 {
		response.WriteError(w, http.StatusBadRequest, "invalid_request", "at least one uploaded file is required")
		return
	}

	staged, err := stageFilesystemUpload(files)
	if err != nil {
		if errors.Is(err, errInvalidUploadPath) {
			response.WriteError(w, http.StatusBadRequest, "invalid_upload_path", err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "upload_failed", err.Error())
		return
	}

	metadata := map[string]string{
		metadataFilesystemUploadID:    staged.UploadID,
		metadataFilesystemUploadRoot:  staged.Root,
		metadataFilesystemUploadFiles: strconv.Itoa(staged.FileCount),
	}
	if staged.OriginalName != "" {
		metadata[metadataFilesystemUploadName] = staged.OriginalName
	}

	writeSourceIngest(w, r, filesystemsource.NewConnector(), sourceIngestInput{
		URI:      staged.IngestURI,
		Metadata: metadata,
	})
}

func uploadedFileEntries(form *multipart.Form) []uploadFileEntry {
	if form == nil {
		return nil
	}
	if files := form.File["files"]; len(files) > 0 {
		paths := form.Value["paths"]
		entries := make([]uploadFileEntry, 0, len(files))
		for index, file := range files {
			relativePath := file.Filename
			if index < len(paths) && strings.TrimSpace(paths[index]) != "" {
				relativePath = paths[index]
			}
			entries = append(entries, uploadFileEntry{File: file, RelativePath: relativePath})
		}
		return entries
	}

	keys := make([]string, 0, len(form.File))
	for key := range form.File {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	entries := make([]uploadFileEntry, 0)
	for _, key := range keys {
		fieldFiles := form.File[key]
		sort.Slice(fieldFiles, func(i, j int) bool {
			return fieldFiles[i].Filename < fieldFiles[j].Filename
		})
		for _, file := range fieldFiles {
			entries = append(entries, uploadFileEntry{File: file, RelativePath: file.Filename})
		}
	}
	return entries
}

func stageFilesystemUpload(files []uploadFileEntry) (stagedFilesystemUpload, error) {
	uploadID, err := newFilesystemUploadID()
	if err != nil {
		return stagedFilesystemUpload{}, err
	}

	root := filepath.Join(filesystemUploadRoot(), uploadID)
	relativePaths := make([]string, 0, len(files))
	seen := map[string]struct{}{}
	for _, file := range files {
		relativePath, err := cleanUploadRelativePath(file.RelativePath)
		if err != nil {
			_ = os.RemoveAll(root)
			return stagedFilesystemUpload{}, err
		}
		if _, exists := seen[relativePath]; exists {
			_ = os.RemoveAll(root)
			return stagedFilesystemUpload{}, fmt.Errorf("%w: duplicate path %q", errInvalidUploadPath, relativePath)
		}
		seen[relativePath] = struct{}{}

		target, err := uploadTargetPath(root, relativePath)
		if err != nil {
			_ = os.RemoveAll(root)
			return stagedFilesystemUpload{}, err
		}
		if err := copyUploadedFile(file.File, target); err != nil {
			_ = os.RemoveAll(root)
			return stagedFilesystemUpload{}, err
		}
		relativePaths = append(relativePaths, relativePath)
	}

	ingestURI := root
	originalName := ""
	if len(relativePaths) == 1 {
		originalName = relativePaths[0]
		if !strings.Contains(relativePaths[0], "/") {
			ingestURI = filepath.Join(root, filepath.FromSlash(relativePaths[0]))
		}
	}

	return stagedFilesystemUpload{
		UploadID:     uploadID,
		Root:         root,
		IngestURI:    ingestURI,
		FileCount:    len(relativePaths),
		OriginalName: originalName,
	}, nil
}

func filesystemUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("FILESYSTEM_UPLOAD_ROOT")); root != "" {
		return root
	}
	return filesystemUploadRootDefault
}

func newFilesystemUploadID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func cleanUploadRelativePath(filename string) (string, error) {
	name := strings.TrimSpace(strings.ReplaceAll(filename, "\\", "/"))
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("%w: empty filename", errInvalidUploadPath)
	}
	if path.IsAbs(name) || looksLikeWindowsAbsolutePath(name) {
		return "", fmt.Errorf("%w: absolute path %q", errInvalidUploadPath, filename)
	}

	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: traversal path %q", errInvalidUploadPath, filename)
	}
	return cleaned, nil
}

func looksLikeWindowsAbsolutePath(name string) bool {
	return len(name) >= 3 && name[1] == ':' && (name[2] == '/' || name[2] == '\\')
}

func uploadTargetPath(root, relativePath string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs == rootAbs || !strings.HasPrefix(targetAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: path escapes upload root", errInvalidUploadPath)
	}
	return target, nil
}

func copyUploadedFile(file *multipart.FileHeader, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	in, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
