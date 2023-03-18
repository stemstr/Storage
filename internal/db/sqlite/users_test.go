package sqlite

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUser(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := CreateUserRequest{
		Pubkey: "123",
	}

	result, err := r.CreateUser(context.TODO(), req)
	assert.Nil(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, req.Pubkey, result.Pubkey)
}

func TestGetUser(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	const pubkey = "xxxxxxxx"
	req := CreateUserRequest{
		Pubkey: pubkey,
	}

	_, err = r.CreateUser(context.TODO(), req)
	assert.Nil(t, err)

	user, err := r.GetUser(context.TODO(), pubkey)
	assert.Nil(t, err)
	assert.Equal(t, pubkey, user.Pubkey)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.CreatedAt)
}

func TestListUsers(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	pubkeys := []string{"xxxxxxxx", "yyyyyyy", "zzzzzz"}

	for _, pubkey := range pubkeys {
		_, err = r.CreateUser(context.TODO(), CreateUserRequest{pubkey})
		assert.Nil(t, err)
	}

	users, err := r.ListUsers(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, users, len(pubkeys))

	var gotPubkeys []string
	for _, user := range users {
		gotPubkeys = append(gotPubkeys, user.Pubkey)
	}
	sort.Strings(gotPubkeys)
	assert.Equal(t, pubkeys, gotPubkeys)
}
