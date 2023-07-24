package subscription

import (
	"context"
)

type mockSubscriptionRepo struct {
	CreateSubscriptionSub    *Subscription
	CreateSubscriptionErr    error
	GetActiveSubscriptionSub *Subscription
	GetActiveSubscriptionErr error
	UpdateSubscriptionErr    error
}

func (m *mockSubscriptionRepo) CreateSubscription(ctx context.Context, sub Subscription) (*Subscription, error) {
	return m.CreateSubscriptionSub, m.CreateSubscriptionErr
}
func (m *mockSubscriptionRepo) GetActiveSubscription(ctx context.Context, pubkey string) (*Subscription, error) {
	return m.GetActiveSubscriptionSub, m.GetActiveSubscriptionErr
}
func (m *mockSubscriptionRepo) UpdateSubscription(ctx context.Context, sub Subscription) error {
	return m.UpdateSubscriptionErr
}

type mockLNProvider struct {
	CreateInvoiceInvoice *Invoice
	CreateInvoiceErr     error
	IsInvoicePaidBool    bool
	IsInvoicePaidErr     error
}

func (m *mockLNProvider) CreateInvoice(ctx context.Context, sub Subscription) (*Invoice, error) {
	return m.CreateInvoiceInvoice, m.CreateInvoiceErr
}
func (m *mockLNProvider) IsInvoicePaid(ctx context.Context, id string) (bool, error) {
	return m.IsInvoicePaidBool, m.IsInvoicePaidErr
}
