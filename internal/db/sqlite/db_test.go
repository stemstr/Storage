package sqlite

import (
	"os"
	"testing"
)

func TestNewRepo(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)
	_, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
}
