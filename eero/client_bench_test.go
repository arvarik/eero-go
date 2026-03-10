package eero

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type DummyData struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

var largePayload []byte
var badPayload []byte

func init() {
	largePayload = []byte(`{"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"}, "data": {`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			largePayload = append(largePayload, []byte(`, `)...)
		}
		largePayload = append(largePayload, []byte(`"dummy_`)...)
		largePayload = append(largePayload, []byte(string(rune(i)))...)
		largePayload = append(largePayload, []byte(`": "hello"`)...)
	}
	largePayload = append(largePayload, []byte(`, "foo": "hello", "bar": 42}`)...)

	badPayload = []byte(`{"meta": {"code": 200, "server_time": "2023-10-01T12:00:00Z"}, "data": {`)
	for i := 0; i < 1000; i++ {
		if i > 0 {
			badPayload = append(badPayload, []byte(`, `)...)
		}
		badPayload = append(badPayload, []byte(`"dummy_`)...)
		badPayload = append(badPayload, []byte(string(rune(i)))...)
		badPayload = append(badPayload, []byte(`": "hello"`)...)
	}
	badPayload = append(badPayload, []byte(`, "foo": "hello", "bar": 42`)...) // missing closing brace to make it invalid JSON
}

func BenchmarkDo(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largePayload)
	}))
	defer server.Close()

	client, _ := NewClient()
	client.BaseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := client.newRequest(ctx, "GET", "/test", nil)
		var out DummyData
		_ = client.do(req, &out)
	}
}

func BenchmarkDoRaw(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(largePayload)
	}))
	defer server.Close()

	client, _ := NewClient()
	client.BaseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := client.newRequest(ctx, "GET", "/test", nil)
		var out EeroResponse[DummyData]
		_ = client.doRaw(req, &out)
	}
}

func BenchmarkDoParseError(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(badPayload)
	}))
	defer server.Close()

	client, _ := NewClient()
	client.BaseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := client.newRequest(ctx, "GET", "/test", nil)
		var out DummyData
		_ = client.do(req, &out)
	}
}

func BenchmarkDoRawParseError(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(badPayload)
	}))
	defer server.Close()

	client, _ := NewClient()
	client.BaseURL = server.URL

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := client.newRequest(ctx, "GET", "/test", nil)
		var out EeroResponse[DummyData]
		_ = client.doRaw(req, &out)
	}
}
