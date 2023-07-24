package subscription

import "errors"

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionExpired  = errors.New("subscription expired")
	ErrSubscriptionUnpaid   = errors.New("subscription unpaid")
)
