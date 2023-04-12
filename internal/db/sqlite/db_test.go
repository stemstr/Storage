package sqlite

import (
	"path/filepath"
	"testing"
)

func TestNewRepo(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "tmp.db")
	_, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
}
