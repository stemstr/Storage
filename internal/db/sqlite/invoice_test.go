package sqlite

import (
	"context"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateInvoice(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := CreateInvoiceRequest{
		PaymentHash: "123",
		Sats:        5000,
		Provider:    "test",
		CreatedBy:   "yyyyyy",
	}

	result, err := r.CreateInvoice(context.TODO(), req)
	assert.Nil(t, err)
	assert.NotNil(t, result.ID)
	assert.Equal(t, req.CreatedBy, result.CreatedBy)
	assert.Equal(t, req.PaymentHash, result.PaymentHash)
	assert.Equal(t, req.Sats, result.Sats)
	assert.Equal(t, false, result.Paid)
	assert.NotEmpty(t, result.CreatedAt)
}

func TestGetInvoice(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	req := CreateInvoiceRequest{
		PaymentHash: "123",
		Sats:        1233,
		Provider:    "test",
		CreatedBy:   "yyyyyy",
	}

	m, err := r.CreateInvoice(context.TODO(), req)
	assert.Nil(t, err)

	Invoice, err := r.GetInvoice(context.TODO(), m.ID)
	assert.Nil(t, err)
	assert.Equal(t, req.CreatedBy, Invoice.CreatedBy)
	assert.NotEmpty(t, Invoice.ID)
	assert.NotEmpty(t, Invoice.CreatedAt)
}

func TestListInvoices(t *testing.T) {
	const testDB = "./tmp.db"
	defer os.Remove(testDB)

	r, err := New(testDB)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	pubkeys := []string{"xxxxxxxx", "yyyyyyy", "zzzzzz"}

	for _, pubkey := range pubkeys {
		_, err = r.CreateInvoice(context.TODO(), CreateInvoiceRequest{
			PaymentHash: "123",
			Sats:        1231,
			Provider:    "test",
			CreatedBy:   pubkey,
		})
		assert.Nil(t, err)
	}

	invoices, err := r.ListInvoices(context.TODO())
	assert.Nil(t, err)
	assert.Len(t, invoices, len(pubkeys))

	var gotPubkeys []string
	for _, inv := range invoices {
		gotPubkeys = append(gotPubkeys, inv.CreatedBy)
	}
	sort.Strings(gotPubkeys)
	assert.Equal(t, pubkeys, gotPubkeys)
}
