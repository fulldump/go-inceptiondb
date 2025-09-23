package inceptiondb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnline(t *testing.T) {
	t.SkipNow()
	c, err := NewClient("https://inceptiondb.io")
	if err != nil {
		t.Fatal(err)
	}
	cols, err := c.ListCollections(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(cols)
}

func TestEncodeQueryRequestNil(t *testing.T) {
	reader, err := encodeQueryRequest((*FindRequest)(nil))
	if err != nil {
		t.Fatalf("encodeQueryRequest() error = %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if got := string(data); got != "{}" {
		t.Fatalf("encodeQueryRequest() = %s, want {}", got)
	}
}

func TestEncodeQueryRequestPayload(t *testing.T) {
	req := &FindRequest{
		QueryOptions: QueryOptions{
			Index:  "my-index",
			Limit:  5,
			Filter: map[string]any{"name": "Fulanez"},
		},
	}
	reader, err := encodeQueryRequest(req)
	if err != nil {
		t.Fatalf("encodeQueryRequest() error = %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got := payload["index"]; got != req.Index {
		t.Fatalf("index = %v, want %s", got, req.Index)
	}
	if got := payload["limit"]; got != float64(req.Limit) {
		t.Fatalf("limit = %v, want %d", got, req.Limit)
	}
	if _, ok := payload["filter"].(map[string]any); !ok {
		t.Fatalf("filter type = %T, want map[string]any", payload["filter"])
	}
}

func TestClientWithAPICredentials(t *testing.T) {
	const (
		apiKey    = "test-key"
		apiSecret = "test-secret"
	)

	var receivedKey, receivedSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Api-Key")
		receivedSecret = r.Header.Get("Api-Secret")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "[]")
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, WithAPICredentials(apiKey, apiSecret))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}

	if receivedKey != apiKey {
		t.Fatalf("Api-Key header = %q, want %q", receivedKey, apiKey)
	}
	if receivedSecret != apiSecret {
		t.Fatalf("Api-Secret header = %q, want %q", receivedSecret, apiSecret)
	}
}

func TestClientWithoutAPICredentials(t *testing.T) {
	var receivedKey, receivedSecret string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("Api-Key")
		receivedSecret = r.Header.Get("Api-Secret")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "[]")
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if _, err := client.ListCollections(context.Background()); err != nil {
		t.Fatalf("ListCollections() error = %v", err)
	}

	if receivedKey != "" {
		t.Fatalf("Api-Key header = %q, want empty", receivedKey)
	}
	if receivedSecret != "" {
		t.Fatalf("Api-Secret header = %q, want empty", receivedSecret)
	}
}
