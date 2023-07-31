package nodeless

import (
	"context"

	"github.com/nodeless-io/go-nodeless"

	sub "github.com/stemstr/storage/internal/subscription"
)

func New(apiKey, storeID string, testnet bool) (*Client, error) {
	c, err := nodeless.New(nodeless.Config{
		APIKey:     apiKey,
		UseTestnet: testnet,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		Client:  c,
		storeID: storeID,
	}, nil
}

type Client struct {
	*nodeless.Client
	storeID string
}

func (c *Client) CreateInvoice(ctx context.Context, s sub.Subscription) (*sub.Invoice, error) {
	invoice, err := c.CreateStoreInvoice(ctx, nodeless.CreateInvoiceRequest{
		StoreID:  c.storeID,
		Amount:   float64(s.Sats),
		Currency: "SATS",
	})
	if err != nil {
		return nil, err
	}

	return &sub.Invoice{
		ID:               invoice.ID,
		LightningInvoice: invoice.LightningInvoice,
	}, nil
}

func (c *Client) IsInvoicePaid(ctx context.Context, id string) (bool, error) {
	status, err := c.GetStoreInvoiceStatus(ctx, c.storeID, id)
	if err != nil {
		return false, err
	}

	if status == nodeless.InvoiceStatusPaid {
		return true, nil
	}

	// TODO: Determine how to handle expired invoices

	return false, nil
}
