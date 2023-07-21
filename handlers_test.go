package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPubkeyIsAllowed(t *testing.T) {
	var tests = []struct {
		name     string
		pubkeys  []string
		pubkey   string
		expected bool
	}{
		{"default open", []string{}, "123", true},
		{"whitelisted", []string{"123"}, "123", true},
		{"whitelisted", []string{"123", "456"}, "123", true},
		{"not whitelisted", []string{"123", "456"}, "789", false},
	}

	for _, tt := range tests {
		result := pubkeyIsAllowed(tt.pubkeys, tt.pubkey)
		assert.Equal(t, tt.expected, result)
	}
}
