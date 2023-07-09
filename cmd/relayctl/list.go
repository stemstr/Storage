package main

import (
	"fmt"
	"log"

	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

var (
	listSrcType  string
	listSrcURL   string
	listByID     bool
	listByPubkey bool
)

func init() {
	listCmd.Flags().StringVarP(&listSrcType, "src", "", "", "list source type: sqlite3 or postgresql")
	listCmd.Flags().StringVarP(&listSrcURL, "srcurl", "", "", "list source url: /path/to/relay.db or postgresql://...")
	listCmd.Flags().BoolVarP(&listByID, "note", "", false, "list by note id")
	listCmd.Flags().BoolVarP(&listByPubkey, "pubkey", "", false, "list by pubkey")

	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "list events from the relay database",
	RunE:  doList,
}

func doList(cmd *cobra.Command, args []string) error {
	if listByID && listByPubkey {
		return fmt.Errorf("can only list by note or pubkey, not both")
	}

	backend, err := initBackend(listSrcType, listSrcURL)
	if err != nil {
		return err
	}

	filter := nostr.Filter{}
	if listByID {
		filter.IDs = args
	}
	if listByPubkey {
		filter.Authors = args
	}

	events, err := backend.QueryEvents(&filter)
	if err != nil {
		return fmt.Errorf("query events first page: %w", err)
	}

	fmt.Printf("ID\tPubkey\tContent\n")

	for len(events) > 0 {
		for _, event := range events {
			fmt.Printf("%s\t%s\t%s\n", event.ID, event.PubKey, event.Content)
		}

		// Load next page
		oldest := events[len(events)-1].CreatedAt
		filter.Until = &oldest
		events, err = backend.QueryEvents(&filter)
		if err != nil {
			log.Printf("[error] query events until %s: %s\n", oldest, err)
		}
	}

	return nil
}
