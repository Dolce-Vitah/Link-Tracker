package grpcscrapper

import (
	"encoding/json"
	"sync"

	"google.golang.org/grpc/encoding"
)

type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
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
