package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

type nostrProvider interface {
	Publish(context.Context, nostr.Event) nostr.Status
}

func connectToRelay(ctx context.Context, url string) (*nostr.Relay, error) {
	return nostr.RelayConnect(ctx, url)
}

func newAudioEvent(pk, desc string, tags []string, streamURL, downloadURL string) nostr.Event {
	var hashTags nostr.Tags
	for _, tag := range tags {
		hashTags = append(hashTags, []string{"t", tag})
	}
	return nostr.Event{
		PubKey:    pk,
		CreatedAt: time.Now(),
		Kind:      nostr.KindTextNote,
		Tags: append(hashTags,
			nostr.Tag{"stream_url", streamURL},
			nostr.Tag{"download_url", downloadURL},
		),
		Content: desc,
	}
}

func parseEncodedEvent(e string) (*nostr.Event, error) {
	eventBytes, err := base64.StdEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}

	var event nostr.Event
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		return nil, fmt.Errorf("must provide valid event")
	}

	return &event, nil
}
