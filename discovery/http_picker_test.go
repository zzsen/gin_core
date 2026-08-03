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

type stubWaitPicker struct {
	stubPicker
	waitInst Instance
	waitErr  error
	ready    chan struct{}
}

func (s *stubWaitPicker) PickWait(ctx context.Context, service, strategy string) (Instance, error) {
	if s.ready != nil {
		select {
		case <-s.ready:
		case <-ctx.Done():
			return Instance{}, ctx.Err()
		}
	}
	if s.waitErr != nil {
		return Instance{}, s.waitErr
	}
	if s.waitInst.IP != "" {
		return s.waitInst, nil
	}
	return s.Pick(service, strategy)
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

// TestHTTPPicker_DoWait_RequiresWaitPicker 非 WaitPicker 报错
func TestHTTPPicker_DoWait_RequiresWaitPicker(t *testing.T) {
	h := NewHTTPPicker(&stubPicker{err: ErrNoInstances})
	_, err := h.DoWait(context.Background(), "svc", http.MethodGet, "/", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PickWait")
}

// TestHTTPPicker_DoWait成功 等待后请求成功
func TestHTTPPicker_DoWait成功(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()
	u, err := url.Parse(ts.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)

	ready := make(chan struct{})
	p := &stubWaitPicker{
		stubPicker: stubPicker{err: ErrNoInstances},
		waitInst:   Instance{IP: u.Hostname(), Port: port, Weight: 1},
		ready:      ready,
	}
	h := NewHTTPPicker(p)
	go func() {
		close(ready)
	}()
	resp, err := h.DoWait(context.Background(), "svc", http.MethodGet, "/", nil, nil)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(t, 200, resp.StatusCode)
}
