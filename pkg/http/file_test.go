package http

import (
	"bytes"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileUpload(t *testing.T) {
	t.Parallel()

	body := &bytes.Buffer{}
	writer := createMultipartForm(body, map[string]string{
		"field":    "value",
		"filename": "test.txt",
		"content":  "Hello, World!",
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

	if string(content) != "Hello, World!" {
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
