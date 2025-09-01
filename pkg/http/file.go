package http

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/shuldan/framework/pkg/contracts"
)

type FileUpload struct {
	ctx    *httpContext
	parsed bool
	form   *multipart.Form
	logger contracts.Logger
}

func (f *FileUpload) Parse(maxMemory int64) error {
	if f.parsed {
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

func (f *FileUpload) FormFile(name string) (contracts.HTTPFile, error) {
	if !f.parsed {
		return nil, ErrMustCallParse
	}

	if f.form == nil || f.form.File == nil {
		return nil, ErrFileNotFound.WithDetail("name", name)
	}

	files, exists := f.form.File[name]
	if !exists || len(files) == 0 {
		return nil, ErrFileNotFound.WithDetail("name", name)
	}

	return &HTTPFileImpl{header: files[0], logger: f.logger}, nil
}

func (f *FileUpload) FormFiles(name string) ([]contracts.HTTPFile, error) {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil { // 32 MB default
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
		files[i] = &HTTPFileImpl{header: header, logger: f.logger}
	}

	return files, nil
}

func (f *FileUpload) FormValue(name string) string {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil { // 32 MB default
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

func (f *FileUpload) FormValues(name string) []string {
	if !f.parsed {
		if err := f.Parse(32 << 20); err != nil { // 32 MB default
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

type HTTPFileImpl struct {
	header *multipart.FileHeader
	logger contracts.Logger
}

func (f *HTTPFileImpl) Header() map[string][]string {
	return map[string][]string(f.header.Header)
}

func (f *HTTPFileImpl) Filename() string {
	return f.header.Filename
}

func (f *HTTPFileImpl) Size() int64 {
	return f.header.Size
}

func (f *HTTPFileImpl) Open() (io.ReadCloser, error) {
	return f.header.Open()
}

func (f *HTTPFileImpl) Save(path string) error {
	if !isPathSafe("", path) {
		return errors.New("invalid file path")
	}

	file, err := f.Open()
	if err != nil {
		return err
	}
	defer func(file io.ReadCloser) {
		if err := file.Close(); err != nil && f.logger != nil {
			f.logger.Error("Failed to close uploaded file", "error", err)
		}
	}(file)

	dst, err := os.Create(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer func(dst *os.File) {
		if err := dst.Close(); err != nil && f.logger != nil {
			f.logger.Error("Failed to close destination file", "error", err)
		}
	}(dst)

	_, err = io.Copy(dst, file)
	return err
}
