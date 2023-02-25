package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nbd-wtf/go-nostr"
)

const (
	// TODO: Determine custom event kind to use
	audioEventKind = 10666
)

type nostrProvider interface {
	Publish(context.Context, nostr.Event) nostr.Status
}

func connectToRelay(ctx context.Context, url string) (*nostr.Relay, error) {
	return nostr.RelayConnect(ctx, url)
}

func newAudioEvent(pk, desc string, tags []string, streamURL, downloadURL string) nostr.Event {
	data, _ := json.Marshal(struct {
		Tags        []string `json:"tags"`
		Desc        string   `json:"desc"`
		StreamURL   string   `json:"stream_url"`
		DownloadURL string   `json:"download_url"`
	}{tags, desc, streamURL, downloadURL})

	return nostr.Event{
		PubKey:    pk,
		CreatedAt: time.Now(),
		Kind:      audioEventKind,
		Tags:      nil,
		Content:   string(data),
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
