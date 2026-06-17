// Package traefik_query_to_header is a Traefik middleware plugin that
// copies values from incoming request query parameters into HTTP request
// headers before the request reaches the next handler in the chain.
package traefik_query_to_header

import (
	"context"
	"fmt"
	"net/http"
)

// Mapping describes a single query parameter to header translation.
type Mapping struct {
	// Query is the name of the query parameter to read.
	Query string `json:"query,omitempty"`
	// Header is the name of the HTTP header to set with the query
	// parameter's value.
	Header string `json:"header,omitempty"`
	// Remove, when true, strips the query parameter from the request
	// URL after it has been copied to the header.
	Remove bool `json:"remove,omitempty"`
	// Overwrite, when true, replaces an existing header with the same
	// name. When false, the header is only set if it is not already
	// present on the request.
	Overwrite bool `json:"overwrite,omitempty"`
	// Bearer, when true, prefixes the header value with "Bearer ".
	Bearer bool `json:"bearer,omitempty"`
}

// Config holds the plugin configuration loaded from the Traefik dynamic
// configuration.
type Config struct {
	Mappings []Mapping `json:"mappings,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// QueryToHeader is the middleware handler.
type QueryToHeader struct {
	next     http.Handler
	mappings []Mapping
	name     string
}

// New creates a new QueryToHeader middleware instance.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	for i, m := range config.Mappings {
		if m.Query == "" {
			return nil, fmt.Errorf("mapping %d: query name must not be empty", i)
		}
		if m.Header == "" {
			return nil, fmt.Errorf("mapping %d: header name must not be empty", i)
		}
	}

	return &QueryToHeader{
		next:     next,
		mappings: config.Mappings,
		name:     name,
	}, nil
}

// ServeHTTP applies the configured query-to-header mappings and forwards
// the request to the next handler.
func (q *QueryToHeader) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	for _, m := range q.mappings {
		values, ok := query[m.Query]
		if !ok || len(values) == 0 {
			continue
		}

		if !m.Overwrite && req.Header.Get(m.Header) != "" {
			continue
		}

		value := values[0]
		if m.Bearer {
			value = "Bearer " + value
		}

		req.Header.Set(m.Header, value)

		if m.Remove {
			query.Del(m.Query)
		}
	}

	req.URL.RawQuery = query.Encode()

	q.next.ServeHTTP(rw, req)
}
