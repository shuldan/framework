package http

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpFileUpload struct {
	ctx    *httpContext
	parsed bool
	form   *multipart.Form
	logger contracts.Logger
}

func (f *httpFileUpload) Parse(maxMemory int64) error {
	if f.parsed {
		return nil
	}

	contentType := f.ctx.req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		f.form = &multipart.Form{
			Value: make(map[string][]string),
			File:  make(map[string][]*multipart.FileHeader),
		}
		f.parsed = true
		return nil
	}

	err := f.ctx.req.ParseMultipartForm(maxMemory)
	if err != nil {
		return ErrFormParse.WithCause(err)
	}
	f.form = f.ctx.req.MultipartForm
	f.parsed = true
	return nil
}

func (f *httpFileUpload) FormFile(name string) (contracts.HTTPFile, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrFileNotFound.WithDetail("name", name).WithDetail("reason", "empty name")
	}

	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil {
			return nil, err
		}
	}

	if f.form == nil || f.form.File == nil {
		return nil, ErrFileNotFound.WithDetail("name", name).WithDetail("reason", "no files in form")
	}

	files, exists := f.form.File[name]
	if !exists || len(files) == 0 {
		return nil, ErrFileNotFound.WithDetail("name", name).WithDetail("reason", "file not found in form")
	}

	return &httpFileImpl{header: files[0], logger: f.logger}, nil
}

func (f *httpFileUpload) FormFiles(name string) ([]contracts.HTTPFile, error) {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil {
			return nil, err
		}
	}

	if f.form == nil || f.form.File == nil {
		return []contracts.HTTPFile{}, nil
	}

	headers, exists := f.form.File[name]
	if !exists {
		return []contracts.HTTPFile{}, nil
	}

	files := make([]contracts.HTTPFile, len(headers))
	for i, header := range headers {
		files[i] = &httpFileImpl{header: header, logger: f.logger}
	}

	return files, nil
}

func (f *httpFileUpload) FormValue(name string) string {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil {
			return ""
		}
	}
	if f.form == nil || f.form.Value == nil {
		return ""
	}
	values, exists := f.form.Value[name]
	if !exists || len(values) == 0 {
		return ""
	}
	return values[0]
}

func (f *httpFileUpload) FormValues(name string) []string {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil {
			return []string{}
		}
	}
	if f.form == nil || f.form.Value == nil {
		return []string{}
	}
	values, exists := f.form.Value[name]
	if !exists {
		return []string{}
	}
	return values
}

type httpFileImpl struct {
	header *multipart.FileHeader
	logger contracts.Logger
}

func (f *httpFileImpl) Header() map[string][]string {
	return map[string][]string(f.header.Header)
}

func (f *httpFileImpl) Filename() string {
	return f.header.Filename
}

func (f *httpFileImpl) Size() int64 {
	return f.header.Size
}

func (f *httpFileImpl) Open() (io.ReadCloser, error) {
	return f.header.Open()
}

func (f *httpFileImpl) Save(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("path cannot be empty")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	file, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && f.logger != nil {
			f.logger.Error("Failed to close uploaded file", "error", closeErr)
		}
	}()
	cleanPath := filepath.Clean(path)
	dst, err := os.OpenFile(cleanPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil && f.logger != nil {
			f.logger.Error("Failed to close destination file", "error", closeErr)
		}
	}()
	const maxFileSize = 100 << 20
	_, err = io.CopyN(dst, file, maxFileSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to copy file data: %w", err)
	}
	return nil
}
