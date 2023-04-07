package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fiatjaf/relayer"
	"github.com/fiatjaf/relayer/storage/sqlite3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
)

const (
	stemstrKindAudio = 1808
)

func newRelay(port int, dbFile, infoPubkey, infoContact, infoDesc, infoVersion string) *Relay {
	return &Relay{
		port:        port,
		storage:     &sqlite3.SQLite3Backend{DatabaseURL: dbFile},
		updates:     make(chan nostr.Event),
		infoPubkey:  infoPubkey,
		infoContact: infoContact,
		infoDesc:    infoDesc,
		infoVersion: infoVersion,
	}
}

type Relay struct {
	port        int
	storage     *sqlite3.SQLite3Backend
	updates     chan nostr.Event
	infoPubkey  string
	infoContact string
	infoDesc    string
	infoVersion string
}

func (r *Relay) GetNIP11InformationDocument() nip11.RelayInformationDocument {
	supportedNIPs := []int{9, 11, 12, 15, 16, 20}
	return nip11.RelayInformationDocument{
		Name:          r.Name(),
		Description:   r.infoDesc,
		PubKey:        r.infoPubkey,
		Contact:       r.infoContact,
		SupportedNIPs: supportedNIPs,
		Software:      "https://github.com/Stemstr",
		Version:       r.infoVersion,
	}
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

	switch evt.Kind {
	case // Allow the following events
		stemstrKindAudio,
		nostr.KindSetMetadata,
		nostr.KindTextNote,
		nostr.KindContactList,
		nostr.KindBoost,
		nostr.KindReaction:
	default: // Reject all others
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
