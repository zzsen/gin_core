package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPicker struct {
	inst Instance
	err  error
}

func (s *stubPicker) Pick(service, strategy string) (Instance, error) {
	if s.err != nil {
		return Instance{}, s.err
	}
	return s.inst, nil
}

// TestHTTPPicker_空实例立即错误
func TestHTTPPicker_空实例立即错误(t *testing.T) {
	p := &stubPicker{err: ErrNoInstances}
	h := NewHTTPPicker(p)
	_, err := h.Do(context.Background(), "svc", http.MethodGet, "/x", nil, nil)
	require.ErrorIs(t, err, ErrNoInstances)
}

// TestHTTPPicker_Do成功
func TestHTTPPicker_Do成功(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	p := &stubPicker{inst: Instance{IP: u.Hostname(), Port: port}}
	h := NewHTTPPicker(p)
	resp, err := h.Do(context.Background(), "svc", http.MethodGet, "/", nil, nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)
}
