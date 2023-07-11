package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fiatjaf/relayer/v2"
	"github.com/fiatjaf/relayer/v2/storage/postgresql"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip11"
)

var defaultAllowedKinds = []int{
	nostr.KindSetMetadata,
	nostr.KindTextNote,
	nostr.KindContactList,
	nostr.KindBoost,
	nostr.KindReaction,
	1808,  // Stemstr audio note
	30078, // NIP-78
	1063,  // NIP-94
}

type relayConfig struct {
	Port             int
	DatabaseURL      string
	AllowedKinds     []int
	Nip11Pubkey      string
	Nip11Contact     string
	Nip11Description string
	Nip11Version     string
}

// func newRelay(port int, databaseURL, infoPubkey, infoContact, infoDesc, infoVersion string) (*Relay, error) {
func newRelay(cfg relayConfig) (*Relay, error) {
	if len(cfg.AllowedKinds) == 0 {
		cfg.AllowedKinds = defaultAllowedKinds
	}

	r := Relay{
		cfg: cfg,
		storage: &postgresql.PostgresBackend{
			DatabaseURL:       cfg.DatabaseURL,
			QueryLimit:        1000,
			QueryAuthorsLimit: 1000,
			QueryIDsLimit:     1000,
			QueryKindsLimit:   10,
			QueryTagsLimit:    20,
		},
		updates: make(chan nostr.Event),
	}

	if err := r.storage.Init(); err != nil {
		return nil, fmt.Errorf("relay init: %w", err)
	}

	return &r, nil
}

type Relay struct {
	cfg     relayConfig
	storage *postgresql.PostgresBackend
	updates chan nostr.Event
}

func (r *Relay) GetNIP11InformationDocument() nip11.RelayInformationDocument {
	return nip11.RelayInformationDocument{
		Name:          r.Name(),
		Description:   r.cfg.Nip11Description,
		PubKey:        r.cfg.Nip11Pubkey,
		Contact:       r.cfg.Nip11Contact,
		SupportedNIPs: []int{9, 11, 12, 15, 16, 20, 78, 94},
		Software:      "https://github.com/Stemstr",
		Version:       r.cfg.Nip11Version,
	}
}

func (r Relay) Name() string {
	return "Stemstr relay"
}

func (r Relay) Storage(ctx context.Context) relayer.Storage {
	return r.storage
}

func (r Relay) OnInitialized(*relayer.Server) {}

func (r Relay) Init() error {
	return nil
}

func (r Relay) AcceptEvent(ctx context.Context, evt *nostr.Event) bool {
	// block events that are too large
	jsonb, _ := json.Marshal(evt)
	if len(jsonb) > 10000 {
		return false
	}

	kindAllowed := false
	for _, kind := range r.cfg.AllowedKinds {
		if evt.Kind == kind {
			kindAllowed = true
			break
		}
	}

	if !kindAllowed {
		return false
	}

	fmt.Printf("relay: received event: %v\n", string(jsonb))

	return true
}

func (relay Relay) InjectEvents() chan nostr.Event {
	return relay.updates
}

func (r Relay) Start() error {
	server, err := relayer.NewServer(r)
	if err != nil {
		return fmt.Errorf("relayer new server: %w", err)
	}

	return server.Start("0.0.0.0", r.cfg.Port)
}
