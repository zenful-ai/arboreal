package arboreal

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// recordingRoundTripper is a base transport that records that it ran and the
// Authorization header it received, then delegates to http.DefaultTransport so
// the real request still completes. It proves NewBearerTransport composes onto
// an existing transport rather than replacing it.
type recordingRoundTripper struct {
	called  bool
	gotAuth string
}

func (rt *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.called = true
	rt.gotAuth = req.Header.Get("Authorization")
	return http.DefaultTransport.RoundTrip(req)
}

// TestNewBearerTransport_WrapsBaseAndAddsHeader covers the one non-obvious
// behavior of this code: NewBearerTransport layers onto an existing transport
// rather than replacing it.
func TestNewBearerTransport_WrapsBaseAndAddsHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Authorization", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	spy := &recordingRoundTripper{}

	rt, err := NewBearerTransport("secret-token", spy)
	if err != nil {
		t.Fatalf("NewBearerTransport returned error: %v", err)
	}

	resp, err := (&http.Client{Transport: rt}).Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// The wrapped base transport ran (composition, not replacement)...
	if !spy.called {
		t.Fatal("base transport was not called; NewBearerTransport did not delegate to it")
	}
	// ...and it saw the request with the bearer header already attached.
	if spy.gotAuth != "Bearer secret-token" {
		t.Fatalf("base transport saw Authorization %q, want %q", spy.gotAuth, "Bearer secret-token")
	}
	// And the header reached the server end to end.
	if got := resp.Header.Get("X-Echo-Authorization"); got != "Bearer secret-token" {
		t.Fatalf("server saw Authorization %q, want %q", got, "Bearer secret-token")
	}
}
