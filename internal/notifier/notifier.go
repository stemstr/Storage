package notifier

import (
	"context"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func New(nsec string) *Notifier {
	_, sk, err := nip19.Decode(nsec)
	if err != nil {
		panic(fmt.Errorf("nip19 decode: %w", err))
	}
	privateKey := sk.(string)

	pubkey, err := nostr.GetPublicKey(privateKey)
	if err != nil {
		panic(fmt.Errorf("get pubkey: %w", err))
	}

	npub, err := nip19.EncodePublicKey(pubkey)
	if err != nil {
		panic(fmt.Errorf("encode pubkey: %w", err))
	}

	return &Notifier{
		relayURLs:  []string{"wss://nostr.mutinywallet.com"},
		npub:       npub,
		pubkey:     pubkey,
		privateKey: privateKey,
	}
}

type Notifier struct {
	relayURLs                []string
	npub, pubkey, privateKey string
}

func (n *Notifier) Send(ctx context.Context, content string) {
	event := n.newEvent(content)
	n.connectAndSend(ctx, event)
}

func (n *Notifier) newEvent(content string) nostr.Event {
	event := nostr.Event{
		PubKey:    n.pubkey,
		CreatedAt: nostr.Now(),
		Kind:      nostr.KindTextNote,
		Tags:      nil,
		Content:   content,
	}
	event.Sign(n.privateKey)

	return event
}

func (n *Notifier) connectAndSend(ctx context.Context, event nostr.Event) {
	for _, url := range n.relayURLs {
		relay, err := nostr.RelayConnect(ctx, url)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer relay.Close()

		_, err = relay.Publish(ctx, event)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}
