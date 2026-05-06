package webhooks

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostTimesOut(t *testing.T) {
	original := dispatchTimeout
	dispatchTimeout = 200 * time.Millisecond
	defer func() { dispatchTimeout = original }()

	const serverDelay = 600 * time.Millisecond

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(serverDelay)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	start := time.Now()
	post(srv.URL, []byte(`{"event":"test","subject":"/software"}`), "")
	elapsed := time.Since(start)

	require.Less(t, elapsed, serverDelay-100*time.Millisecond,
		"post should time out before the %s server sleep; took %s", serverDelay, elapsed)

	assert.GreaterOrEqual(t, elapsed, dispatchTimeout-50*time.Millisecond,
		"post should have waited at least the timeout duration")
}
