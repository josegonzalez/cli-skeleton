package command

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientUserAgent(t *testing.T) {
	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer server.Close()

	m := &Meta{appName: "hello-world", version: "1.2.3"}
	req, err := m.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := m.HTTPClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got != "hello-world/1.2.3" {
		t.Fatalf("User-Agent = %q", got)
	}

	req, _ = m.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("User-Agent", "custom")
	resp, err = m.HTTPClient().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "custom" {
		t.Fatalf("an explicit User-Agent should be kept: %q", got)
	}

	if (&Meta{}).UserAgent() != "cli/dev" {
		t.Fatalf("default User-Agent = %q", (&Meta{}).UserAgent())
	}
}

func TestHTTPClientTimeout(t *testing.T) {
	m := &Meta{}
	if got := m.HTTPClient().Timeout; got != DefaultTimeout {
		t.Errorf("default timeout = %v", got)
	}

	d := time.Minute
	m = &Meta{defaultTimeout: &d}
	if got := m.HTTPClient().Timeout; got != time.Minute {
		t.Errorf("app default timeout = %v", got)
	}

	m = &Meta{}
	f := m.FlagSet("fetch", FlagSetTimeout)
	if err := f.Parse([]string{"--timeout", "0"}); err != nil {
		t.Fatal(err)
	}
	if got := m.HTTPClient().Timeout; got != 0 {
		t.Errorf("--timeout 0 should disable the timeout: %v", got)
	}
}

func TestHTTPClientProxy(t *testing.T) {
	// net/http reads HTTP_PROXY, HTTPS_PROXY and NO_PROXY once per process,
	// so the test only checks that the client uses the environment.
	transport := (&Meta{}).HTTPClient().Transport.(*userAgentTransport).base.(*http.Transport)
	if transport.Proxy == nil {
		t.Fatal("the client should use the proxy from the environment")
	}

	req, _ := http.NewRequest(http.MethodGet, "https://api.example.com", nil)
	if _, err := transport.Proxy(req); err != nil {
		t.Fatal(err)
	}
}

func TestNewRequestUsesCommandContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m := &Meta{Context: ctx}
	req, err := m.NewRequest(http.MethodGet, "http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.HTTPClient().Do(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
