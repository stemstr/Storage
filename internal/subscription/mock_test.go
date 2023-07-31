package subscription

import (
	"context"
)

type mockSubscriptionRepo struct {
	CreateSubscriptionSub     *Subscription
	CreateSubscriptionErr     error
	GetActiveSubscriptionSubs []Subscription
	GetActiveSubscriptionErr  error
	UpdateSubscriptionErr     error
}

func (m *mockSubscriptionRepo) CreateSubscription(ctx context.Context, sub Subscription) (*Subscription, error) {
	return m.CreateSubscriptionSub, m.CreateSubscriptionErr
}
func (m *mockSubscriptionRepo) GetActiveSubscriptions(ctx context.Context, pubkey string) ([]Subscription, error) {
	return m.GetActiveSubscriptionSubs, m.GetActiveSubscriptionErr
}
func (m *mockSubscriptionRepo) UpdateStatus(ctx context.Context, id int64, status SubscriptionStatus) error {
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
