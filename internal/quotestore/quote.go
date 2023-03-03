package quotestore

import (
	"time"
)

type Request struct {
	Pubkey      string
	ContentHash string
	ContentSize int
}

type Quote struct {
	ID          string
	Pubkey      string
	ContentHash string
	ContentSize int
	PaymentHash *string
	Paid        bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
