package grpcserver

import (
	"encoding/json"
	"fmt"
	"sync"

	"google.golang.org/grpc/encoding"
)

type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal grpc json payload: %w", err)
	}
	return data, nil
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("unmarshal grpc json payload: %w", err)
	}
	return nil
}

func (JSONCodec) Name() string {
	return "json"
}

var registerCodecOnce sync.Once

func RegisterJSONCodec() {
	registerCodecOnce.Do(func() {
		encoding.RegisterCodec(JSONCodec{})
	})
}
