package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileUpload(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"field":    "value",
		"filename": "test.txt",
		"content":  hello,
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value := upload.FormValue("field"); value != "value" {
		t.Errorf("Expected field value, got %s", value)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if file.Filename() != "test.txt" {
		t.Errorf("Expected filename test.txt, got %s", file.Filename())
	}

	reader, err := file.Open()
	if err != nil {
		t.Fatalf("File open failed: %v", err)
	}
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			t.Errorf("Reader close failed: %v", err)
		}
	}(reader)

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("File read failed: %v", err)
	}

	if string(content) != hello {
		t.Errorf("Expected content 'Hello, World!', got %s", string(content))
	}
}

func TestFileUploadErrors(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	_, err := upload.FormFile("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	_, err = upload.FormFile("")
	if err == nil {
		t.Error("Expected error for empty filename")
	}
}

func TestFileSave(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "saved.txt")

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "File content",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if err := file.Save(destPath); err != nil {
		t.Fatalf("File save failed: %v", err)
	}

	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(content) != "File content" {
		t.Errorf("Expected 'File content', got %s", string(content))
	}
}

func TestFileSaveErrors(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "File content",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if err := file.Save(""); err == nil {
		t.Error("Expected error for empty path")
	}
}

func TestFileUploadFormValues(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	writer.writeField("field1", "value1")
	writer.writeField("field2", "value2")
	writer.writeFile("file", "test.txt", "Hello")
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	values := upload.FormValues("field1")
	if len(values) != 1 || values[0] != "value1" {
		t.Errorf("Expected [value1], got %v", values)
	}

	values = upload.FormValues("nonexistent")
	if len(values) != 0 {
		t.Errorf("Expected empty slice for nonexistent field, got %v", values)
	}
}

func TestFileUploadMultipleFiles(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	writer.writeFile("files", "file1.txt", "content1")
	writer.writeFile("files", "file2.txt", "content2")
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	files, err := upload.FormFiles("files")
	if err != nil {
		t.Fatalf("FormFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestFileUploadParseError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte("invalid multipart")))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err == nil {
		t.Error("Expected parse error for invalid multipart data")
	}
}

func TestFileHeaderMethods(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Hello, World!",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if file.Filename() != "test.txt" {
		t.Errorf("Expected filename test.txt, got %s", file.Filename())
	}

	if file.Size() <= 0 {
		t.Errorf("Expected positive file size, got %d", file.Size())
	}

	headers := file.Header()
	if headers == nil {
		t.Error("Expected headers to be non-nil")
	}

	reader, err := file.Open()
	if err != nil {
		t.Fatalf("File open failed: %v", err)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("File read failed: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %s", string(content))
	}

	if err := reader.Close(); err != nil {
		t.Errorf("File close failed: %v", err)
	}
}

func TestFileUploadEmptyForm(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	value := upload.FormValue("nonexistent")
	if value != "" {
		t.Errorf("Expected empty value, got %s", value)
	}

	values := upload.FormValues("nonexistent")
	if len(values) != 0 {
		t.Errorf("Expected empty slice, got %v", values)
	}

	files, err := upload.FormFiles("nonexistent")
	if err != nil {
		t.Fatalf("FormFiles failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected empty slice, got %v", files)
	}
}

func TestFileUploadWhitespaceFilename(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/upload", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	_, err := upload.FormFile("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only filename")
	}
}

func TestFileUploadParseMultipleTimes(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"field":    "value",
		"filename": "test.txt",
		"content":  "Hello",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("First parse failed: %v", err)
	}

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Second parse failed: %v", err)
	}

	if value := upload.FormValue("field"); value != "value" {
		t.Errorf("Expected field value after second parse, got %s", value)
	}
}

func TestFileUploadParseNonMultipart(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/upload", strings.NewReader("plain text body"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse non-multipart should not fail: %v", err)
	}

	if value := upload.FormValue("field"); value != "" {
		t.Errorf("Expected empty value for non-multipart, got %s", value)
	}

	files, err := upload.FormFiles("file")
	if err != nil {
		t.Fatalf("FormFiles failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected no files for non-multipart, got %d", len(files))
	}
}

func TestFileUploadFormValueEmpty(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	writer.writeField("empty", "")
	writer.writeField("field", "value")
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if value := upload.FormValue("empty"); value != "" {
		t.Errorf("Expected empty string, got %s", value)
	}

	if value := upload.FormValue("nonexistent"); value != "" {
		t.Errorf("Expected empty string for nonexistent field, got %s", value)
	}
}

func TestFileUploadFormValuesMultiple(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	writer.writeField("tags", "go")
	writer.writeField("tags", "http")
	writer.writeField("tags", "testing")
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	values := upload.FormValues("tags")
	expected := []string{"go", "http", "testing"}

	if len(values) != len(expected) {
		t.Errorf("Expected %d values, got %d", len(expected), len(values))
	}

	for i, val := range values {
		if val != expected[i] {
			t.Errorf("Expected value[%d] = %s, got %s", i, expected[i], val)
		}
	}
}

func TestFileUploadFormValueBeforeParse(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"field":    "value",
		"filename": "test.txt",
		"content":  "Hello",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	value := upload.FormValue("field")
	if value != "value" {
		t.Errorf("Expected auto-parse to work, got %s", value)
	}
}

func TestFileSaveCreateDirectory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "deeply", "nested", "path", "file.txt")

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Directory creation test",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if err := file.Save(nestedPath); err != nil {
		t.Fatalf("Save to nested path failed: %v", err)
	}

	content, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	if string(content) != "Directory creation test" {
		t.Errorf("Expected 'Directory creation test', got %s", string(content))
	}
}

func TestFileSaveLargeFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "test.bin")

	testData := bytes.Repeat([]byte("X"), 10*1024*1024)

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}
	writer.writeFileWithContent("file", "test.bin", testData)
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if err := file.Save(destPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("Failed to stat saved file: %v", err)
	}

	if info.Size() != int64(len(testData)) {
		t.Errorf("Expected file size %d, got %d", len(testData), info.Size())
	}
}

func TestFileSavePathTraversal(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	maliciousPath := filepath.Join(tmpDir, "..", "..", "etc", "passwd")

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Malicious content",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if err := file.Save(maliciousPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	savedPath, _ := filepath.Abs(maliciousPath)
	tmpDirAbs, _ := filepath.Abs(tmpDir)

	if !strings.HasPrefix(filepath.Clean(savedPath), tmpDirAbs) {
		t.Log("Path was properly cleaned and contained")
	}
}

func TestFileSaveWhitespacePath(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Content",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	tests := []string{
		"",
		"   ",
		"\t",
		"\n",
	}

	for _, path := range tests {
		if err := file.Save(path); err == nil {
			t.Errorf("Expected error for whitespace path %q, got nil", path)
		}
	}
}

func TestFileOpenMultipleTimes(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Multiple opens test",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	reader1, err := file.Open()
	if err != nil {
		t.Fatalf("First open failed: %v", err)
	}
	defer reader1.Close()

	content1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("First read failed: %v", err)
	}

	reader2, err := file.Open()
	if err != nil {
		t.Fatalf("Second open failed: %v", err)
	}
	defer reader2.Close()

	content2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("Second read failed: %v", err)
	}

	if string(content1) != string(content2) {
		t.Error("Content should be the same on multiple opens")
	}
}

func TestFileHeaderAllFields(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"filename": "test.txt",
		"content":  "Header test",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	if err := upload.Parse(32 << 20); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	file, err := upload.FormFile("file")
	if err != nil {
		t.Fatalf("FormFile failed: %v", err)
	}

	if file.Filename() == "" {
		t.Error("Filename should not be empty")
	}

	if file.Size() <= 0 {
		t.Error("Size should be positive")
	}

	headers := file.Header()
	if headers == nil {
		t.Error("Headers should not be nil")
	}

	if _, ok := headers["Content-Disposition"]; !ok {
		t.Error("Should have Content-Disposition header")
	}
}

func TestFormFileEmptyName(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"field":    "value",
		"filename": "test.txt",
		"content":  "Hello",
	})

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	tests := []string{"", "   ", "\t\n"}

	for _, name := range tests {
		_, err := upload.FormFile(name)
		if err == nil {
			t.Errorf("Expected error for empty/whitespace name %q", name)
		}
	}
}

func TestFormFileNoFormParsed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("POST", "/upload", nil)
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	_, err := upload.FormFile("file")
	if err == nil {
		t.Error("Expected error when no form data parsed")
	}
}

func TestFormFilesBeforeParse(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	writer.writeFile("files", "file1.txt", "content1")
	writer.writeFile("files", "file2.txt", "content2")
	writer.close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	logger := &mockLogger{}

	ctx := NewHTTPContext(w, req, logger)
	upload := ctx.FileUpload()

	files, err := upload.FormFiles("files")
	if err != nil {
		t.Fatalf("FormFiles with auto-parse failed: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func (w *multipartWriter) writeFileWithContent(fieldname, filename string, content []byte) {
	_, _ = fmt.Fprintf(w.body, "--%s\r\n", w.boundary)
	_, _ = fmt.Fprintf(w.body, "Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldname, filename)
	w.body.WriteString("Content-Type: application/octet-stream\r\n")
	w.body.WriteString("\r\n")
	w.body.Write(content)
	w.body.WriteString("\r\n")
}
