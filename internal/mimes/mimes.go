package mimes

import (
	"strings"
)

type mime struct {
	Mimetype        string
	Extension       string
	OtherMimetypes  []string
	OtherExtensions []string
}

var supportedMimes = []mime{
	{
		Mimetype:        "audio/aiff",
		Extension:       ".aif",
		OtherMimetypes:  []string{"audio/x-aiff"},
		OtherExtensions: []string{".aiff"},
	},
	{
		Mimetype:       "audio/flac",
		Extension:      ".flac",
		OtherMimetypes: []string{"audio/x-flac"},
	},
	{
		Mimetype:       "audio/mp3",
		Extension:      ".mp3",
		OtherMimetypes: []string{"audio/mpeg3", "audio/x-mpeg-3", "audio/mpeg"},
	},
	{
		Mimetype:       "audio/mp4",
		Extension:      ".m4a",
		OtherMimetypes: []string{"audio/m4a"},
	},
	{
		Mimetype:       "audio/ogg",
		Extension:      ".ogg",
		OtherMimetypes: []string{"audio/ogg"},
	},
	{
		Mimetype:       "audio/wave",
		Extension:      ".wav",
		OtherMimetypes: []string{"audio/wav", "audio/x-wav"},
	},
}

// FileExtension returns a file extension for a mimetype.
func FileExtension(mimetype string) string {
	for _, supported := range supportedMimes {
		if strings.EqualFold(supported.Mimetype, mimetype) || contains(supported.OtherMimetypes, mimetype) {
			return supported.Extension
		}
	}

	return ""
}

// FromFilename returns a mimetype for a filename based on file extension.
func FromFilename(name string) string {
	for _, supported := range supportedMimes {
		if strings.HasSuffix(name, supported.Extension) {
			return supported.Mimetype
		}
		for _, otherExtension := range supported.OtherExtensions {
			if strings.HasSuffix(name, otherExtension) {
				return supported.Mimetype
			}
		}
	}

	return ""
}

func contains[T comparable](arr []T, v T) bool {
	for _, el := range arr {
		if el == v {
			return true
		}
	}
	return false
}
