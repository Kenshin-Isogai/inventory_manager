package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"

	gcs "cloud.google.com/go/storage"
)

type Store interface {
	Save(ctx context.Context, key string, body io.Reader) (string, error)
	Delete(ctx context.Context, path string) error
}

func New(cfg config.StorageConfig) (Store, error) {
	switch cfg.Mode {
	case "local":
		return NewLocal(cfg.Artifacts)
	case "cloud":
		return NewCloudStore(cfg.BucketName)
	default:
		return nil, fmt.Errorf("unsupported storage mode: %s", cfg.Mode)
	}
}

func ReadAll(ctx context.Context, path string) ([]byte, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("artifact path is required")
	}
	if !strings.HasPrefix(path, "gs://") {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read local artifact: %w", err)
		}
		return data, nil
	}

	bucket, objectKey, err := parseCloudPath("", path)
	if err != nil {
		return nil, err
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cloud storage client: %w", err)
	}
	defer client.Close()

	reader, err := client.Bucket(bucket).Object(objectKey).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open cloud artifact reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read cloud artifact: %w", err)
	}
	return data, nil
}

type LocalStore struct {
	baseDir string
}

func NewLocal(baseDir string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create artifacts dir: %w", err)
	}
	return &LocalStore{baseDir: baseDir}, nil
}

func (s *LocalStore) Save(_ context.Context, key string, body io.Reader) (string, error) {
	fullPath := filepath.Join(s.baseDir, filepath.Clean(strings.TrimPrefix(key, "/")))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("create parent dir: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fullPath, nil
}

func (s *LocalStore) Delete(_ context.Context, path string) error {
	if path == "" {
		return nil
	}

	cleanPath := filepath.Clean(path)
	relPath, err := filepath.Rel(filepath.Clean(s.baseDir), cleanPath)
	if err != nil {
		return fmt.Errorf("resolve artifact path: %w", err)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("artifact path is outside base dir: %s", path)
	}

	if err := os.Remove(cleanPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove artifact: %w", err)
	}

	parentDir := filepath.Dir(cleanPath)
	for parentDir != "." && parentDir != string(filepath.Separator) {
		if parentDir == filepath.Clean(s.baseDir) {
			break
		}
		if err := os.Remove(parentDir); err != nil {
			break
		}
		parentDir = filepath.Dir(parentDir)
	}

	return nil
}

type CloudStore struct {
	bucket string
}

func NewCloudStore(bucket string) (*CloudStore, error) {
	if strings.TrimSpace(bucket) == "" {
		return nil, fmt.Errorf("cloud storage bucket is required in cloud mode")
	}
	return &CloudStore{bucket: bucket}, nil
}

func (s *CloudStore) Save(ctx context.Context, key string, body io.Reader) (string, error) {
	client, err := gcs.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("create cloud storage client: %w", err)
	}
	defer client.Close()

	objectKey := normalizeCloudKey(key)
	writer := client.Bucket(s.bucket).Object(objectKey).NewWriter(ctx)
	if _, err := io.Copy(writer, body); err != nil {
		_ = writer.Close()
		return "", fmt.Errorf("write cloud artifact: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close cloud artifact writer: %w", err)
	}
	return fmt.Sprintf("gs://%s/%s", s.bucket, objectKey), nil
}

func (s *CloudStore) Delete(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	bucket, objectKey, err := parseCloudPath(s.bucket, path)
	if err != nil {
		return err
	}

	client, err := gcs.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("create cloud storage client: %w", err)
	}
	defer client.Close()

	if err := client.Bucket(bucket).Object(objectKey).Delete(ctx); err != nil && err != gcs.ErrObjectNotExist {
		return fmt.Errorf("delete cloud artifact: %w", err)
	}
	return nil
}

func normalizeCloudKey(key string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(strings.TrimSpace(key))), "/")
}

func parseCloudPath(defaultBucket, path string) (string, string, error) {
	if strings.HasPrefix(path, "gs://") {
		trimmed := strings.TrimPrefix(path, "gs://")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			return "", "", fmt.Errorf("invalid gs path: %s", path)
		}
		return parts[0], parts[1], nil
	}
	if strings.TrimSpace(defaultBucket) == "" {
		return "", "", fmt.Errorf("cloud storage bucket is required to delete path %s", path)
	}
	return defaultBucket, normalizeCloudKey(path), nil
}
