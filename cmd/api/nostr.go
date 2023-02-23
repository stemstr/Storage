package main

import (
	"context"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const (
	audioEventKind = 10666
)

type nostrProvider interface {
	Publish(context.Context, nostr.Event) nostr.Status
}

func connectToRelay(ctx context.Context, url string) (*nostr.Relay, error) {
	return nostr.RelayConnect(ctx, url)
}

func newAudioEvent(pk, streamURL string) nostr.Event {
	return nostr.Event{
		PubKey:    pk,
		CreatedAt: time.Now(),
		Kind:      audioEventKind,
		Tags:      nil,
		Content:   streamURL,
	}
}
