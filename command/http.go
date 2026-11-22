package command

import (
	"io"
	"net/http"
	"time"
)

// HTTPClient returns an HTTP client for the command. It honors
// HTTP_PROXY, HTTPS_PROXY and NO_PROXY, times out according to --timeout
// (or the default timeout when the command has no --timeout flag), and
// identifies itself as `<app>/<version>`.
//
// Use NewRequest to build requests that are cancelled when the user
// interrupts the command.
func (m *Meta) HTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyFromEnvironment

	return &http.Client{
		Transport: &userAgentTransport{base: transport, userAgent: m.UserAgent()},
		Timeout:   m.httpTimeout(),
	}
}

// httpTimeout returns the timeout for HTTP requests.
func (m *Meta) httpTimeout() time.Duration {
	if m.flagBits&FlagSetTimeout != 0 {
		return m.flags.timeout
	}
	if m.defaultTimeout != nil {
		return *m.defaultTimeout
	}
	return DefaultTimeout
}

// NewRequest returns an HTTP request bound to the command context, so it is
// cancelled when the user interrupts the command.
func (m *Meta) NewRequest(method string, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(m.context(), method, url, body)
}

// UserAgent returns the User-Agent sent by HTTPClient: `<app>/<version>`.
func (m *Meta) UserAgent() string {
	name := m.appName
	if name == "" {
		name = "cli"
	}
	version := m.version
	if version == "" {
		version = "dev"
	}
	return name + "/" + version
}

// userAgentTransport sets the User-Agent header on requests that do not set
// one.
type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req = req.Clone(req.Context())
		req.Header.Set("User-Agent", t.userAgent)
	}
	return t.base.RoundTrip(req)
}
