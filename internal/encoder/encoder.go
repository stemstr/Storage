package encoder

import (
	"context"
	"errors"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
)

type Encoder interface {
	Encode(context.Context, EncodeRequest) (EncodeResponse, error)
}

type EncodeRequest struct {
	InputPath  string
	InputType  string
	OutputDir  string
	OutputName string
}

type EncodeResponse struct {
	Output string
}
