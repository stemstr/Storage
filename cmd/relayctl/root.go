package main

import (
	"github.com/spf13/cobra"
)

var (
	verbose bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

var rootCmd = &cobra.Command{
	Use:   "relayctl",
	Short: "stemstr relay CLI",
	Run: func(cmd *cobra.Command, args []string) {
	},
}
