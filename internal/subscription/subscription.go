package subscription

import (
	"context"
	"fmt"
	"time"
)

func New(repo subscriptionRepo, ln lnProvider) (*SubscriptionService, error) {
	return &SubscriptionService{
		repo: repo,
		ln:   ln,
	}, nil
}

type SubscriptionService struct {
	repo subscriptionRepo
	ln   lnProvider
}

type subscriptionRepo interface {
	CreateSubscription(ctx context.Context, sub Subscription) (*Subscription, error)
	GetActiveSubscription(ctx context.Context, pubkey string) (*Subscription, error)
	UpdateStatus(ctx context.Context, id int64, status SubscriptionStatus) error
}

type lnProvider interface {
	CreateInvoice(ctx context.Context, sub Subscription) (*Invoice, error)
	IsInvoicePaid(ctx context.Context, id string) (bool, error)
}

// GetActiveSubscription fetches the active subscription for a pubkey.
// An error is returned if the subscription is not found, unpaid, or expired.
func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, pubkey string) (*Subscription, error) {
	sub, err := s.repo.GetActiveSubscription(ctx, pubkey)
	if err != nil {
		return nil, fmt.Errorf("repo.GetActiveSub: %w", err)
	}

	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}

	// Is the subscription expired?
	if sub.ExpiresAt.Before(time.Now()) {
		return nil, ErrSubscriptionExpired
	}

	if sub.Status == StatusUnpaid {
		// Has the invoice has been paid since we last checked?
		paid, err := s.ln.IsInvoicePaid(ctx, sub.InvoiceID)
		if err != nil {
			return nil, fmt.Errorf("GetLatestSub: %w", err)
		}
		if !paid {
			return sub, ErrSubscriptionUnpaid
		}

		if err := s.repo.UpdateStatus(ctx, sub.ID, StatusPaid); err != nil {
			return nil, fmt.Errorf("UpdateSub: %w", err)
		}

		return sub, nil
	}

	return sub, nil
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, sub Subscription) (*Subscription, error) {
	invoice, err := s.ln.CreateInvoice(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("CreateInvoice: %w", err)
	}

	sub.InvoiceID = invoice.ID
	sub.Status = StatusUnpaid
	sub.LightningInvoice = invoice.LightningInvoice

	newSub, err := s.repo.CreateSubscription(ctx, sub)
	if err != nil {
		return nil, fmt.Errorf("CreateSub: %w", err)
	}

	return newSub, nil
}

type Subscription struct {
	ID               int64              `json:"id" db:"id"`
	Pubkey           string             `json:"pubkey" db:"pubkey"`
	Days             int                `json:"days" db:"days"`
	Sats             int                `json:"sats" db:"sats"`
	InvoiceID        string             `json:"invoice_id" db:"invoice_id"`
	Status           SubscriptionStatus `json:"status" db:"status"`
	LightningInvoice string             `json:"lightning_invoice" db:"lightning_invoice"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at"`
	ExpiresAt        time.Time          `json:"expires_at" db:"expires_at"`
	UpdatedAt        *time.Time         `json:"updated_at" db:"updated_at"`
}

type SubscriptionStatus string

const (
	StatusPaid   SubscriptionStatus = "paid"
	StatusUnpaid SubscriptionStatus = "unpaid"
)

type Invoice struct {
	ID               string `json:"id"`
	LightningInvoice string `json:"lightning_invoice"`
	QRCode           string `json:"qr_code"`
}
