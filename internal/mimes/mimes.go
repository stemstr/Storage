package mimes

import (
	"strings"
)

const (
	AudioAIFF = "audio/aiff"
	AudioFLAC = "audio/flac"
	AudioMP3  = "audio/mp3"
	AudioMP4  = "audio/mp4"
	AudioOGG  = "audio/ogg"
	AudioWAV  = "audio/wave"
)

func FromFilename(name string) string {
	switch {
	case strings.HasSuffix(name, ".aif"):
		return AudioAIFF
	case strings.HasSuffix(name, ".aiff"):
		return AudioAIFF
	case strings.HasSuffix(name, ".flac"):
		return AudioFLAC
	case strings.HasSuffix(name, ".mp3"):
		return AudioMP3
	case strings.HasSuffix(name, ".m4a"):
		return AudioMP4
	case strings.HasSuffix(name, ".ogg"):
		return AudioOGG
	case strings.HasSuffix(name, ".wav"):
		return AudioWAV
	default:
		return ""
	}
}
