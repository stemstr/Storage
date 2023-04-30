package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fiatjaf/relayer"
	"github.com/fiatjaf/relayer/storage/postgresql"
	"github.com/fiatjaf/relayer/storage/sqlite3"
	"github.com/nbd-wtf/go-nostr"
	"github.com/spf13/cobra"
)

var (
	migrateSrcType  string
	migrateSrcURL   string
	migrateDestType string
	migrateDestURL  string
)

func init() {
	migrateCmd.Flags().StringVarP(&migrateSrcType, "src", "", "", "migration source type: sqlite3 or postgresql")
	migrateCmd.Flags().StringVarP(&migrateSrcURL, "srcurl", "", "", "migration source url: /path/to/relay.db or postgresql://...")
	migrateCmd.Flags().StringVarP(&migrateDestType, "dest", "", "", "migration destination type: sqlite3 or postgresql")
	migrateCmd.Flags().StringVarP(&migrateDestURL, "desturl", "", "", "migration destination url: /path/to/relay.db or postgresql://...")

	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "migrate relay databases",
	RunE:  doMigration,
}

func doMigration(cmd *cobra.Command, args []string) error {
	src, err := initBackend(migrateSrcType, migrateSrcURL)
	if err != nil {
		return err
	}

	dest, err := initBackend(migrateDestType, migrateDestURL)
	if err != nil {
		return err
	}

	result, err := migrate(src, dest)
	if err != nil {
		return err
	}

	log.Printf("completed in %s\ntotal: %d\nerrors: %d\n", result.duration, result.total, result.errors)
	return nil
}

func initBackend(kind, url string) (relayer.Storage, error) {
	var backend relayer.Storage

	switch kind {
	case "postgresql":
		backend = &postgresql.PostgresBackend{DatabaseURL: url}
	case "sqlite3":
		backend = &sqlite3.SQLite3Backend{DatabaseURL: url}
	default:
		return nil, fmt.Errorf("unsupported storage kind %q. must be one of 'postgresql' or 'sqlite3'", kind)
	}

	if err := backend.Init(); err != nil {
		return nil, fmt.Errorf("backend init: %w", err)
	}

	return backend, nil
}

type migrationResult struct {
	total    int
	errors   int
	duration time.Duration
}

func migrate(src, dest relayer.Storage) (*migrationResult, error) {
	start := time.Now()

	result := &migrationResult{}
	defer func(start time.Time) {
		result.duration = time.Since(start)
	}(start)

	// Load the most recent page
	events, err := src.QueryEvents(&nostr.Filter{})
	if err != nil {
		return result, fmt.Errorf("query events first page: %w", err)
	}

	for len(events) > 0 {
		// Save every event
		for _, event := range events {
			if verbose {
				jsonb, _ := json.Marshal(event)
				log.Printf("event: %s\n", string(jsonb))
			}

			if err := dest.SaveEvent(&event); err != nil {
				result.errors += 1
				log.Printf("[error] SaveEvent %s: %s\n", event.ID, err)
			} else {
				result.total += 1
				log.Printf("saved %s\n", event.ID)
			}
		}

		// Load next page
		oldest := events[len(events)-1].CreatedAt
		events, err = src.QueryEvents(&nostr.Filter{
			Until: &oldest,
		})
		if err != nil {
			log.Printf("[error] query events until %s: %s\n", oldest, err)
		}
	}

	return result, nil
}
