package subscription

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetActiveSubscription(t *testing.T) {
	var tests = []struct {
		name   string
		repo   subscriptionRepo
		ln     lnProvider
		pubkey string
		sub    *Subscription
		err    error
	}{
		{
			name: "active paid subscription",
			repo: &mockSubscriptionRepo{
				GetActiveSubscriptionSub: &Subscription{
					ID:        123,
					Pubkey:    "xxx",
					Days:      30,
					Sats:      5000,
					InvoiceID: "lnxxx",
					Status:    StatusPaid,
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(time.Hour * 24),
				},
			},
			ln:     &mockLNProvider{},
			pubkey: "xxx",
			sub: &Subscription{
				ID:        123,
				Pubkey:    "xxx",
				Days:      30,
				Sats:      5000,
				InvoiceID: "lnxxx",
				Status:    StatusPaid,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		},
		{
			name: "expired subscription",
			repo: &mockSubscriptionRepo{
				GetActiveSubscriptionSub: &Subscription{
					ID:        123,
					Pubkey:    "xxx",
					Days:      30,
					Sats:      5000,
					InvoiceID: "lnxxx",
					Status:    StatusPaid,
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(time.Hour * -24),
				},
			},
			ln:     &mockLNProvider{},
			pubkey: "xxx",
			err:    ErrSubscriptionExpired,
		},
		{
			name: "unpaid subscription",
			repo: &mockSubscriptionRepo{
				GetActiveSubscriptionSub: &Subscription{
					ID:               123,
					Pubkey:           "xxx",
					Days:             30,
					Sats:             5000,
					InvoiceID:        "uuid",
					Status:           StatusUnpaid,
					LightningInvoice: "lnbc123",
					CreatedAt:        time.Now(),
					ExpiresAt:        time.Now().Add(time.Hour * 24),
				},
			},
			ln: &mockLNProvider{
				IsInvoicePaidBool: false,
			},
			pubkey: "xxx",
			sub: &Subscription{
				ID:               123,
				Pubkey:           "xxx",
				Days:             30,
				Sats:             5000,
				InvoiceID:        "uuid",
				Status:           StatusUnpaid,
				LightningInvoice: "lnbc123",
				CreatedAt:        time.Now(),
				ExpiresAt:        time.Now().Add(time.Hour * 24),
			},
			err: ErrSubscriptionUnpaid,
		},
		{
			name: "since paid subscription",
			repo: &mockSubscriptionRepo{
				GetActiveSubscriptionSub: &Subscription{
					ID:               123,
					Pubkey:           "xxx",
					Days:             30,
					Sats:             5000,
					InvoiceID:        "uuid",
					Status:           StatusUnpaid,
					LightningInvoice: "lnbc123",
					CreatedAt:        time.Now(),
					ExpiresAt:        time.Now().Add(time.Hour * 24),
				},
			},
			ln: &mockLNProvider{
				IsInvoicePaidBool: true,
			},
			pubkey: "xxx",
			sub: &Subscription{
				ID:               123,
				Pubkey:           "xxx",
				Days:             30,
				Sats:             5000,
				InvoiceID:        "uuid",
				Status:           StatusPaid,
				LightningInvoice: "lnbc123",
				CreatedAt:        time.Now(),
				ExpiresAt:        time.Now().Add(time.Hour * 24),
			},
			err: ErrSubscriptionUnpaid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.repo, tt.ln)
			assert.NoError(t, err)

			ctx := context.Background()
			sub, err := svc.GetActiveSubscription(ctx, tt.pubkey)
			if err != nil {
				assert.Equal(t, tt.err, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.sub.Status, sub.Status)
		})
	}
}

func TestCreateSubscription(t *testing.T) {
	var tests = []struct {
		name   string
		repo   subscriptionRepo
		ln     lnProvider
		pubkey string
		sub    Subscription
		err    error
	}{
		{
			name: "basic subscription",
			repo: &mockSubscriptionRepo{
				CreateSubscriptionSub: &Subscription{
					ID:               123,
					Pubkey:           "xxx",
					Days:             30,
					Sats:             5000,
					InvoiceID:        "lnxxx",
					Status:           StatusUnpaid,
					LightningInvoice: "lnbc123",
					CreatedAt:        time.Now(),
					ExpiresAt:        time.Now().Add(time.Hour * 24),
				},
			},
			ln: &mockLNProvider{
				CreateInvoiceInvoice: &Invoice{
					ID:               "lnxxx",
					LightningInvoice: "lnbc123",
				},
			},
			pubkey: "xxx",
			sub: Subscription{
				ID:        123,
				Pubkey:    "xxx",
				Days:      30,
				Sats:      5000,
				Status:    StatusUnpaid,
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(time.Hour * 24),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			svc, err := New(tt.repo, tt.ln)
			assert.NoError(t, err)

			ctx := context.Background()
			sub, err := svc.CreateSubscription(ctx, tt.sub)
			if err != nil {
				assert.Equal(t, tt.err, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.sub.Status, sub.Status)
			assert.Equal(t, "lnxxx", sub.InvoiceID)
			assert.Equal(t, "lnbc123", sub.LightningInvoice)
		})
	}
}
