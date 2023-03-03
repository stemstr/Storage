package quotestore

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRepo(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)
	_, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateQuote(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := Request{
		Pubkey:      "123",
		ContentHash: "asdkfjsakfsafsaasdf",
		ContentSize: 131241,
	}

	result, err := r.Create(context.TODO(), &req)
	assert.Nil(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, req.Pubkey, result.Pubkey)
	assert.Equal(t, req.ContentHash, result.ContentHash)
	assert.Equal(t, req.ContentSize, result.ContentSize)
}
