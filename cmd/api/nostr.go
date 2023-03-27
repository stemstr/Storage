package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/fiatjaf/relayer"
	"github.com/fiatjaf/relayer/storage/sqlite3"
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

func newRelay(dbFile string, port int) *Relay {
	return &Relay{
		port:    port,
		storage: &sqlite3.SQLite3Backend{DatabaseURL: dbFile},
		updates: make(chan nostr.Event),
	}
}

type Relay struct {
	port    int
	storage *sqlite3.SQLite3Backend
	updates chan nostr.Event
}

func (r *Relay) Name() string {
	return "Stemstr relay"
}

func (r *Relay) Storage() relayer.Storage {
	return r.storage
}

func (r *Relay) OnInitialized(*relayer.Server) {}

func (r *Relay) Init() error {
	return nil
}

func (r *Relay) AcceptEvent(evt *nostr.Event) bool {
	// block events that are too large
	jsonb, _ := json.Marshal(evt)
	if len(jsonb) > 10000 {
		return false
	}

	fmt.Printf("relay: received event: %v\n", string(jsonb))

	return true
}

func (relay *Relay) InjectEvents() chan nostr.Event {
	return relay.updates
}

func (relay *Relay) Publish(ctx context.Context, evt nostr.Event) {
	jsonb, _ := json.Marshal(evt)
	fmt.Printf("relay: inject event: %v\n", string(jsonb))
	if err := relay.storage.SaveEvent(&evt); err != nil {
		fmt.Printf("relay: failed to save event: %v\n", err)
	}
	relay.updates <- evt
}

func (r *Relay) Start() error {
	settings := relayer.Settings{
		Host: "0.0.0.0",
		Port: fmt.Sprintf("%d", r.port),
	}
	return relayer.StartConf(settings, r)
}
