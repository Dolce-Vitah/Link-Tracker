package avrokafka

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/trackerapi"
)

const linkUpdateSchema = `{"type":"record","name":"LinkUpdateEvent","namespace":"com.example.notification","fields":[{"name":"id","type":"long"},{"name":"url","type":"string"},{"name":"description","type":"string"},{"name":"tgChatIds","type":{"type":"array","items":"long"}},{"name":"eventKind","type":"string"},{"name":"title","type":"string"},{"name":"author","type":"string"},{"name":"createdAtUnix","type":"long"},{"name":"preview","type":"string"}]}`

type LinkUpdateCodec struct {
	registryURL string
	subject     string
	client      *http.Client
	mu          sync.RWMutex
	schemaID    int
}

func NewLinkUpdateCodec(registryURL, subject string) (*LinkUpdateCodec, error) {
	if strings.TrimSpace(registryURL) == "" {
		return nil, fmt.Errorf("schema registry URL is required")
	}
	if strings.TrimSpace(subject) == "" {
		return nil, fmt.Errorf("schema subject is required")
	}
	return &LinkUpdateCodec{registryURL: strings.TrimRight(registryURL, "/"), subject: subject, client: &http.Client{Timeout: 5 * time.Second}}, nil
}

func (c *LinkUpdateCodec) Encode(ctx context.Context, u trackerapi.LinkUpdate) ([]byte, error) {
	if err := u.Validate(); err != nil {
		return nil, err
	}
	id, err := c.ensureSchemaID(ctx)
	if err != nil {
		return nil, err
	}
	avro := &bytes.Buffer{}
	encodeLong(avro, u.ID)
	encodeString(avro, u.URL)
	encodeString(avro, u.Description)
	encodeLong(avro, int64(len(u.TgChatIDs)))
	for _, chatID := range u.TgChatIDs {
		encodeLong(avro, chatID)
	}
	encodeLong(avro, 0)
	encodeString(avro, u.EventKind)
	encodeString(avro, u.Title)
	encodeString(avro, u.Author)
	encodeLong(avro, u.CreatedAt.UTC().Unix())
	encodeString(avro, u.Preview)

	out := bytes.NewBuffer(make([]byte, 0, avro.Len()+5))
	out.WriteByte(0)
	_ = binary.Write(out, binary.BigEndian, int32(id))
	out.Write(avro.Bytes())
	return out.Bytes(), nil
}

func (c *LinkUpdateCodec) Decode(payload []byte) (trackerapi.LinkUpdate, error) {
	if len(payload) < 5 || payload[0] != 0 {
		return trackerapi.LinkUpdate{}, fmt.Errorf("invalid avro payload")
	}
	buf := bytes.NewBuffer(payload[5:])
	id, err := decodeLong(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	url, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	description, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	chatIDs, err := decodeLongArray(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	eventKind, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	title, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	author, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	createdAtUnix, err := decodeLong(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}
	preview, err := decodeString(buf)
	if err != nil {
		return trackerapi.LinkUpdate{}, err
	}

	u := trackerapi.NewLinkUpdate(id, url, chatIDs, eventKind, title, author, time.Unix(createdAtUnix, 0).UTC(), preview)
	u.Description = description
	return u, nil
}

func (c *LinkUpdateCodec) ensureSchemaID(ctx context.Context) (int, error) {
	c.mu.RLock()
	if c.schemaID != 0 {
		id := c.schemaID
		c.mu.RUnlock()
		return id, nil
	}
	c.mu.RUnlock()

	body, _ := json.Marshal(map[string]string{"schema": linkUpdateSchema})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.registryURL+"/subjects/"+c.subject+"/versions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/vnd.schemaregistry.v1+json")
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("schema registry status %d: %s", resp.StatusCode, string(b))
	}
	var out struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	c.mu.Lock()
	c.schemaID = out.ID
	c.mu.Unlock()
	return out.ID, nil
}

func encodeLong(buf *bytes.Buffer, n int64) {
	u := uint64((n << 1) ^ (n >> 63))
	for (u & ^uint64(0x7F)) != 0 {
		buf.WriteByte(byte(u&0x7f | 0x80))
		u >>= 7
	}
	buf.WriteByte(byte(u))
}
func decodeLong(buf *bytes.Buffer) (int64, error) {
	var x uint64
	var s uint
	for i := 0; i < 10; i++ {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		if b < 0x80 {
			x |= uint64(b) << s
			n := int64((x >> 1) ^ uint64((int64(x&1)<<63)>>63))
			return n, nil
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, fmt.Errorf("invalid avro long")
}
func encodeString(buf *bytes.Buffer, s string) {
	encodeLong(buf, int64(len([]byte(s))))
	buf.WriteString(s)
}
func decodeString(buf *bytes.Buffer) (string, error) {
	ln, err := decodeLong(buf)
	if err != nil {
		return "", err
	}
	if ln < 0 {
		return "", fmt.Errorf("negative string len")
	}
	b := make([]byte, ln)
	if _, err := io.ReadFull(buf, b); err != nil {
		return "", err
	}
	return string(b), nil
}
func decodeLongArray(buf *bytes.Buffer) ([]int64, error) {
	var out []int64
	for {
		blockCount, err := decodeLong(buf)
		if err != nil {
			return nil, err
		}
		if blockCount == 0 {
			return out, nil
		}
		if blockCount < 0 {
			blockSize, err := decodeLong(buf)
			if err != nil {
				return nil, err
			}
			_ = blockSize
			blockCount = -blockCount
		}
		for i := int64(0); i < blockCount; i++ {
			v, err := decodeLong(buf)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
	}
}
