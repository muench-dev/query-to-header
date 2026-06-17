package query_to_header_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/muench-dev/query-to-header"
)

func TestNew_RejectsNilConfig(t *testing.T) {
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	if _, err := query_to_header.New(context.Background(), next, nil, "test"); err == nil {
		t.Fatal("expected an error for a nil config, got none")
	}
}

func TestNew_RejectsIncompleteMapping(t *testing.T) {
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
	cfg := &query_to_header.Config{
		Mappings: []query_to_header.Mapping{{Query: "token"}},
	}

	if _, err := query_to_header.New(context.Background(), next, cfg, "test"); err == nil {
		t.Fatal("expected an error for a mapping missing a header name, got none")
	}
}

func TestServeHTTP_SetsHeaderFromQuery(t *testing.T) {
	cfg := &query_to_header.Config{
		Mappings: []query_to_header.Mapping{
			{Query: "token", Header: "X-Token"},
		},
	}

	var gotHeader string
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotHeader = req.Header.Get("X-Token")
	})

	handler, err := query_to_header.New(context.Background(), next, cfg, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?token=abc123", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if gotHeader != "abc123" {
		t.Fatalf("expected header value %q, got %q", "abc123", gotHeader)
	}
}

func TestServeHTTP_PrefixesValueWithBearer(t *testing.T) {
	cfg := &query_to_header.Config{
		Mappings: []query_to_header.Mapping{
			{Query: "token", Header: "Authorization", Bearer: true},
		},
	}

	var gotHeader string
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotHeader = req.Header.Get("Authorization")
	})

	handler, err := query_to_header.New(context.Background(), next, cfg, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?token=abc123", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if gotHeader != "Bearer abc123" {
		t.Fatalf("expected header value %q, got %q", "Bearer abc123", gotHeader)
	}
}

func TestServeHTTP_DoesNotOverwriteExistingHeaderByDefault(t *testing.T) {
	cfg := &query_to_header.Config{
		Mappings: []query_to_header.Mapping{
			{Query: "token", Header: "X-Token"},
		},
	}

	var gotHeader string
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotHeader = req.Header.Get("X-Token")
	})

	handler, err := query_to_header.New(context.Background(), next, cfg, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?token=abc123", nil)
	req.Header.Set("X-Token", "preexisting")
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if gotHeader != "preexisting" {
		t.Fatalf("expected header value %q, got %q", "preexisting", gotHeader)
	}
}

func TestServeHTTP_RemovesQueryParamWhenConfigured(t *testing.T) {
	cfg := &query_to_header.Config{
		Mappings: []query_to_header.Mapping{
			{Query: "token", Header: "X-Token", Remove: true},
		},
	}

	var gotQuery string
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		gotQuery = req.URL.RawQuery
	})

	handler, err := query_to_header.New(context.Background(), next, cfg, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/?token=abc123&keep=me", nil)
	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)

	if gotQuery != "keep=me" {
		t.Fatalf("expected remaining query %q, got %q", "keep=me", gotQuery)
	}
}
