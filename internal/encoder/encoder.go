package encoder

import (
	"context"
	"errors"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
)

type Encoder interface {
	HLS(context.Context, EncodeRequest) (EncodeHLSResponse, error)
	WAV(context.Context, EncodeRequest) (EncodeWAVResponse, error)
}

type EncodeRequest struct {
	Mimetype   string
	InputPath  string
	OutputPath string
}

type EncodeHLSResponse struct {
	Output           string
	IndexFilepath    string
	SegmentFilepaths []string
}

type EncodeWAVResponse struct {
	Output   string
	Filepath string
}
