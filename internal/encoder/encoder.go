package encoder

import (
	"context"
	"errors"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
)

type Encoder interface {
	HLS(context.Context, EncodeRequest) (EncodeResponse, error)
	WAV(context.Context, EncodeRequest) (EncodeResponse, error)
}

type EncodeRequest struct {
	Mimetype   string
	InputPath  string
	OutputPath string
}

type EncodeResponse struct {
	Output string
}
