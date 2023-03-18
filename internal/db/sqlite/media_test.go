package sqlite

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateMedia(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := CreateMediaRequest{
		InvoiceID:        "123",
		Size:             78342643,
		Sum:              "23asdfsdf76a5sdfasfd",
		OriginalFilename: "foo.aif",
		CreatedBy:        "yyyyyy",
	}

	result, err := r.CreateMedia(context.TODO(), req)
	assert.Nil(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, req.CreatedBy, result.CreatedBy)
	assert.Equal(t, req.InvoiceID, result.InvoiceID)
	assert.Equal(t, req.Size, result.Size)
	assert.Equal(t, req.Sum, result.Sum)
	assert.Equal(t, req.OriginalFilename, result.OriginalFilename)
	assert.NotEmpty(t, result.CreatedAt)
}

func TestGetMedia(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := CreateMediaRequest{
		InvoiceID:        "123",
		Size:             78342643,
		Sum:              "23asdfsdf76a5sdfasfd",
		OriginalFilename: "foo.aif",
		CreatedBy:        "yyyyyyy",
	}

	m, err := r.CreateMedia(context.TODO(), req)
	assert.Nil(t, err)

	media, err := r.GetMedia(context.TODO(), m.ID)
	assert.Nil(t, err)
	assert.Equal(t, req.CreatedBy, media.CreatedBy)
	assert.NotEmpty(t, media.ID)
	assert.NotEmpty(t, media.CreatedAt)
}

func TestListMedia(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	pubkeys := []string{"xxxxxxxx", "yyyyyyy", "zzzzzz"}

	for _, pubkey := range pubkeys {
		_, err = r.CreateMedia(context.TODO(), CreateMediaRequest{
			InvoiceID:        "123",
			Size:             78342643,
			Sum:              "23asdfsdf76a5sdfasfd",
			OriginalFilename: "foo.aif",
			CreatedBy:        pubkey,
		})
		assert.Nil(t, err)
	}

	media, err := r.ListMedia(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, media, len(pubkeys))

	var gotPubkeys []string
	for _, m := range media {
		gotPubkeys = append(gotPubkeys, m.CreatedBy)
	}
	sort.Strings(gotPubkeys)
	assert.Equal(t, pubkeys, gotPubkeys)
}
