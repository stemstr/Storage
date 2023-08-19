package mock

import (
	"context"

	sub "github.com/stemstr/storage/internal/subscription"
)

func New() *Client {
	return &Client{}
}

type Client struct {
}

func (c *Client) CreateInvoice(ctx context.Context, s sub.Subscription) (*sub.Invoice, error) {
	return &sub.Invoice{
		ID:               "fake",
		LightningInvoice: "lnbcfake",
	}, nil
}

func (c *Client) IsInvoicePaid(ctx context.Context, id string) (bool, error) {
	if id == "paid" {
		return true, nil
	}

	return false, nil
}
