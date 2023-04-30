package main

import (
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

var (
	removeSrcType  string
	removeSrcURL   string
	removeByID     bool
	removeByPubkey bool
)

func init() {
	removeCmd.Flags().StringVarP(&removeSrcType, "src", "", "", "remove source type: sqlite3 or postgresql")
	removeCmd.Flags().StringVarP(&removeSrcURL, "srcurl", "", "", "remove source url: /path/to/relay.db or postgresql://...")
	removeCmd.Flags().BoolVarP(&removeByID, "note", "", false, "remove by note id")
	removeCmd.Flags().BoolVarP(&removeByPubkey, "pubkey", "", false, "remove by pubkey")

	rootCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "rm",
	Short: "remove events from the relay database",
	RunE:  doRemove,
}

func doRemove(cmd *cobra.Command, args []string) error {
	if removeByID && removeByPubkey {
		return fmt.Errorf("can only remove by note or pubkey, not both")
	}
	if !removeByID && !removeByPubkey {
		return fmt.Errorf("must remove by note or pubkey")
	}

	if len(args) == 0 {
		return fmt.Errorf("must provide one or more items to remove")
	}

	backend, err := initBackend(removeSrcType, removeSrcURL)
	if err != nil {
		return err
	}

	filter := nostr.Filter{}
	switch {
	case removeByID:
		filter.IDs = args
	case removeByPubkey:
		filter.Authors = args
	}

	events, err := backend.QueryEvents(&filter)
	if err != nil {
		return fmt.Errorf("query events first page: %w", err)
	}

	total, errors := 0, 0
	for len(events) > 0 {
		// Delete any returned events
		for _, event := range events {
			if err := backend.DeleteEvent(event.ID, event.PubKey); err != nil {
				errors += 1
				log.Printf("[error] DeleteEvent %s: %s\n", event.ID, err)
			} else {
				total += 1
				log.Printf("deleted %s\n", event.ID)
			}
		}

		// Load next page
		oldest := events[len(events)-1].CreatedAt
		filter.Until = &oldest
		events, err = backend.QueryEvents(&filter)
		if err != nil {
			log.Printf("[error] query events until %s: %s\n", oldest, err)
		}
	}

	log.Printf("total: %d\nerrors: %d\n", total, errors)
	return nil
}
