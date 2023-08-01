package zbd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	zebedee "github.com/zebedeeio/go-sdk"

	sub "github.com/stemstr/storage/internal/subscription"
)

func New(apiKey, chargeCallbackURL string) (*Client, error) {
	return &Client{
		Client:            zebedee.New(apiKey),
		chargeCallbackURL: chargeCallbackURL,
	}, nil
}

type Client struct {
	*zebedee.Client
	chargeCallbackURL string
}

func (c *Client) CreateInvoice(ctx context.Context, s sub.Subscription) (*sub.Invoice, error) {
	invoice, err := c.Charge(&zebedee.Charge{
		InternalID:  strconv.Itoa(int(s.ID)),
		Amount:      strconv.Itoa(s.Sats * 1000), // millisats
		Description: fmt.Sprintf("Stemstr %d day subscription", s.Days),
		ExpiresIn:   int64((time.Minute * 5).Seconds()),
		CallbackURL: c.chargeCallbackURL,
	})
	if err != nil {
		return nil, err
	}

	return &sub.Invoice{
		ID:               invoice.ID,
		LightningInvoice: invoice.Invoice.Request,
	}, nil
}

func (c *Client) IsInvoicePaid(ctx context.Context, id string) (bool, error) {
	invoice, err := c.GetCharge(id)
	if err != nil {
		return false, fmt.Errorf("GetCharge: %w", err)
	}

	if invoice.Status == "completed" {
		return true, nil
	}

	return false, nil
}
