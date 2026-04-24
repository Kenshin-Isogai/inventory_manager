package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoreDeleteRemovesArtifactAndEmptyDirectories(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewLocal(baseDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	artifactPath, err := store.Save(context.Background(), filepath.Join("ocr", "job-1", "quote.pdf"), strings.NewReader("pdf"))
	if err != nil {
		t.Fatalf("save artifact: %v", err)
	}

	if err := store.Delete(context.Background(), artifactPath); err != nil {
		t.Fatalf("delete artifact: %v", err)
	}

	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact to be removed, stat err=%v", err)
	}

	jobDir := filepath.Dir(artifactPath)
	if _, err := os.Stat(jobDir); !os.IsNotExist(err) {
		t.Fatalf("expected empty job dir to be removed, stat err=%v", err)
	}
}

func TestLocalStoreDeleteRejectsPathOutsideBaseDir(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewLocal(baseDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	outsidePath := filepath.Join(filepath.Dir(baseDir), "outside.pdf")
	if err := store.Delete(context.Background(), outsidePath); err == nil {
		t.Fatal("expected error for path outside base dir")
	}
}

func TestReadAllReadsLocalArtifact(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	artifactPath := filepath.Join(baseDir, "quote.pdf")
	if err := os.WriteFile(artifactPath, []byte("pdf"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}

	data, err := ReadAll(context.Background(), artifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	if string(data) != "pdf" {
		t.Fatalf("unexpected artifact data: %q", string(data))
	}
}

func TestReadAllRejectsInvalidGSPath(t *testing.T) {
	t.Parallel()

	if _, err := ReadAll(context.Background(), "gs://bucket-only"); err == nil {
		t.Fatal("expected invalid gs path error")
	}
}
