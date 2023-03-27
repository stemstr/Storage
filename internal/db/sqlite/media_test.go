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
		Size:      78342643,
		Sum:       "23asdfsdf76a5sdfasfd",
		Mimetype:  "audio/wav",
		Waveform:  []int{21, 44, 78, 21, 31, 3, 33, 57, 84},
		CreatedBy: "yyyyyy",
	}

	result, err := r.CreateMedia(context.TODO(), req)
	assert.Nil(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, req.CreatedBy, result.CreatedBy)
	assert.Equal(t, req.Size, result.Size)
	assert.Equal(t, req.Sum, result.Sum)
	assert.Equal(t, req.Waveform, result.Waveform)
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
		Size:      78342643,
		Sum:       "23asdfsdf76a5sdfasfd",
		Mimetype:  "audio/wav",
		Waveform:  []int{21, 44, 78, 21, 31, 3, 33, 57, 84},
		CreatedBy: "yyyyyy",
	}

	_, err = r.CreateMedia(context.TODO(), req)
	assert.Nil(t, err)

	media, err := r.GetMedia(context.TODO(), req.Sum)
	assert.Nil(t, err)
	assert.Equal(t, req.CreatedBy, media.CreatedBy)
	assert.NotEmpty(t, media.ID)
	assert.NotEmpty(t, media.Mimetype)
	assert.NotEmpty(t, media.Waveform)
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
			Size:      78342643,
			Sum:       "23asdfsdf76a5sdfasfd",
			Mimetype:  "audio/wav",
			Waveform:  []int{21, 44, 78, 21, 31, 3, 33, 57, 84},
			CreatedBy: pubkey,
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
