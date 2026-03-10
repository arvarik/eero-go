package main

import (
	"encoding/json"
	"testing"
)

var largePayload []byte

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
}

type APIError struct {
	Code int `json:"code"`
}

type DummyData struct {
	Foo string `json:"foo"`
	Bar int    `json:"bar"`
}

func BenchmarkDo_DoubleParse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var meta struct {
			Meta APIError `json:"meta"`
		}
		_ = json.Unmarshal(largePayload, &meta)

		target := &struct {
			Data any `json:"data"`
		}{
			Data: &DummyData{},
		}
		_ = json.Unmarshal(largePayload, target)
	}
}

func BenchmarkDo_SingleParseRawMessage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var combined struct {
			Meta APIError        `json:"meta"`
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(largePayload, &combined)

		var out DummyData
		if len(combined.Data) > 0 && string(combined.Data) != "null" {
			_ = json.Unmarshal(combined.Data, &out)
		}
	}
}
