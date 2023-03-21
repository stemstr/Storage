package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

type nostrProvider interface {
	Publish(context.Context, nostr.Event)
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

func parseTags(tags [][]string) nostr.Tags {
	var t nostr.Tags
	for _, plainTag := range tags {
		var tag nostr.Tag
		for _, tv := range plainTag {
			tag = append(tag, tv)
		}

		t = append(t, tag)
	}
	return t
}
