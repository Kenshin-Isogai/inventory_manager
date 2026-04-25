//go:build integration

package integration

import (
	"os"
	"testing"

	"backend/internal/testutil"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.StopEmbeddedPostgres()
	os.Exit(code)
}
