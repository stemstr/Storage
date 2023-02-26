package main

import (
	"testing"

	"github.com/nbd-wtf/go-nostr"
	"github.com/stretchr/testify/assert"
)

func TestNewAudioEvent(t *testing.T) {
	var tests = []struct {
		name        string
		pk          string
		desc        string
		tags        []string
		streamURL   string
		downloadURL string
	}{
		{
			name:        "no desc or tags",
			pk:          "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
			desc:        "",
			tags:        []string{},
			streamURL:   "https://example.com/stream/123.m3u8",
			downloadURL: "https://example.com/dl/123",
		},
		{
			name:        "desc and tags",
			pk:          "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
			desc:        "test",
			tags:        []string{"hiphop", "soul"},
			streamURL:   "https://example.com/stream/123.m3u8",
			downloadURL: "https://example.com/dl/123",
		},
	}

	getTagValues := func(tags nostr.Tags, key string) []string {
		nTag := tags.GetFirst(nostr.Tag{key})
		if nTag == nil {
			return []string{}
		}

		var vals []string
		for i, val := range *nTag {
			// Skip the tag type.
			if i == 0 {
				continue
			}
			vals = append(vals, val)
		}

		return vals
	}

	getFlattenTagValues := func(tags nostr.Tags, key string) []string {
		nTags := tags.GetAll(nostr.Tag{key})
		if len(nTags) == 0 {
			return []string{}
		}

		var vals []string
		for _, nTag := range nTags {
			for i, val := range nTag {
				// Skip the tag type.
				if i == 0 {
					continue
				}
				vals = append(vals, val)
			}
		}

		return vals
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAudioEvent(tt.pk, tt.desc, tt.tags, tt.streamURL, tt.downloadURL)
			assert.Equal(t, tt.pk, e.PubKey)
			assert.Equal(t, tt.desc, e.Content)
			assert.Equal(t, tt.tags, getFlattenTagValues(e.Tags, "t"))
			assert.Equal(t, []string{tt.streamURL}, getTagValues(e.Tags, "stream_url"))
			assert.Equal(t, []string{tt.downloadURL}, getTagValues(e.Tags, "download_url"))
		})
	}
}
