package pg

import (
	"context"
	"strconv"
	"time"

	sub "github.com/stemstr/storage/internal/subscription"
)

// TODO: implement against postgres. for now it's a dummy inmem for hacking.

func New(dbConnStr string) (*Repo, error) {
	return &Repo{
		m: map[string][]sub.Subscription{},
	}, nil
}

type Repo struct {
	m map[string][]sub.Subscription
}

func (r *Repo) CreateSubscription(ctx context.Context, s sub.Subscription) (*sub.Subscription, error) {
	n := 1
	for _, subs := range r.m {
		n += len(subs)
	}

	s.ID = strconv.Itoa(n)
	subs, ok := r.m[s.Pubkey]
	if !ok {
		subs = []sub.Subscription{s}
	}
	subs = append(subs, s)
	r.m[s.Pubkey] = subs

	return &s, nil
}

func (r *Repo) GetActiveSubscription(ctx context.Context, pubkey string) (*sub.Subscription, error) {
	subs, ok := r.m[pubkey]
	if !ok {
		return nil, nil
	}

	return &subs[len(subs)-1], nil
}

func (r *Repo) UpdateSubscription(ctx context.Context, s sub.Subscription) error {
	subs, ok := r.m[s.Pubkey]
	if !ok {
		return nil
	}

	var updatedSubs []sub.Subscription
	for _, _sub := range subs {
		if _sub.ID == s.ID {
			_sub.Status = s.Status
			_sub.UpdatedAt = time.Now()
		}
		updatedSubs = append(updatedSubs, _sub)
	}

	return nil
}
