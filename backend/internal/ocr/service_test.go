package ocr

import (
	"context"
	"io"
	"testing"
)

type stubStore struct {
	savedPath   string
	deletedPath string
	saveErr     error
	deleteErr   error
}

func (s *stubStore) Save(_ context.Context, key string, _ io.Reader) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	s.savedPath = "artifacts/" + key
	return s.savedPath, nil
}

func (s *stubStore) Delete(_ context.Context, path string) error {
	s.deletedPath = path
	return s.deleteErr
}

type stubProvider struct {
	doc ExtractedDocument
	err error
}

func (p *stubProvider) Extract(context.Context, string, string) (ExtractedDocument, error) {
	if p.err != nil {
		return ExtractedDocument{}, p.err
	}
	return p.doc, nil
}

func (p *stubProvider) Name() string {
	return "stub"
}

func TestCleanupArtifactUsesStoreDelete(t *testing.T) {
	t.Parallel()

	store := &stubStore{}
	service := NewService(nil, store, nil, nil)

	if err := service.cleanupArtifact(context.Background(), "artifacts/ocr/job/file.pdf"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if store.deletedPath != "artifacts/ocr/job/file.pdf" {
		t.Fatalf("unexpected deleted path: %q", store.deletedPath)
	}
}

func TestCleanupArtifactNoopWithoutStore(t *testing.T) {
	t.Parallel()

	service := NewService(nil, nil, nil, nil)
	if err := service.cleanupArtifact(context.Background(), "artifacts/ocr/job/file.pdf"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestScoreSupplierCandidate(t *testing.T) {
	t.Parallel()

	score, reason := scoreSupplierCandidate(normalizeMatchKey("Thorlabs Japan"), "Thorlabs Japan")
	if score < 0.99 {
		t.Fatalf("expected strong match score, got %v", score)
	}
	if reason == "" {
		t.Fatal("expected match reason")
	}
}

func TestScoreItemCandidate(t *testing.T) {
	t.Parallel()

	score, reason := scoreItemCandidate(
		normalizeMatchKey("Phoenix Contact"),
		normalizeMatchKey("MK44-BX"),
		normalizeMatchKey("Terminal block bulk box"),
		"sup-thorlabs",
		itemRecord{
			ItemID:              "item-mk44",
			CanonicalItemNumber: "MK-44",
			Description:         "Terminal block 4P",
			ManufacturerName:    "Phoenix Contact",
			DefaultSupplierID:   "sup-thorlabs",
			SupplierAlias:       "MK44-BX",
		},
	)
	if score < 0.95 {
		t.Fatalf("expected strong item match score, got %v", score)
	}
	if reason == "" {
		t.Fatal("expected match reason")
	}
}
