package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
)

// dispatchTimeout caps the per-request webhook dispatch. It is a var (not
// const) so tests can shorten it without sleeping for whole seconds.
//
//nolint:gochecknoglobals // tunable for tests, effectively const at runtime
var dispatchTimeout = 10 * time.Second

// httpClient is shared across dispatches so the underlying http.Transport
// can reuse TCP and TLS connections to the same subscriber. The per-request
// deadline is enforced via the request context, not via Client.Timeout.
//
//nolint:gochecknoglobals // singleton needed for connection pool reuse
var httpClient = &http.Client{}

func DispatchWebhooks(event models.Event, gorm *gorm.DB) error {
	var webhooks []models.Webhook

	subject := "/" + event.EntityType
	if event.EntityID != "" {
		subject += "/" + event.EntityID
	}

	// When entity_id == '', the webhook is meant for any event occurred in any
	// resource of that type (fe. Publishers, Software)
	stmt := gorm.
		Where(
			"entity_type = ? AND (entity_id = '' OR entity_id = ?)",
			event.EntityType,
			event.EntityID,
		)

	if err := stmt.Select("url, secret").Find(&webhooks).Error; err != nil {
		return fmt.Errorf("error finding webhooks for %s: %w", subject, err)
	}

	jsonBody, err := json.Marshal(map[string]string{
		"event":   event.Type,
		"subject": subject,
	})
	if err != nil {
		return fmt.Errorf("error marshaling event JSON for %s: %w", subject, err)
	}

	for _, webhook := range webhooks {
		signature := ""

		if webhook.Secret != "" {
			h := hmac.New(sha256.New, []byte(webhook.Secret))

			// This can't fail
			_, _ = h.Write(jsonBody)

			signature = hex.EncodeToString(h.Sum(nil))
		}

		go post(webhook.URL, jsonBody, signature)
	}

	return nil
}

func post(url string, body []byte, signature string) {
	ctx, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "DevelopersItaliaAPI-Webhook/1.0")
	req.Header.Set("Content-Type", "application/json")

	if signature != "" {
		req.Header.Set("X-Webhook-Signature", signature)
	}

	response, err := httpClient.Do(req)
	if err != nil {
		//nolint:godox // need to implement this in the future
		// TODO: Replace this and send anonymous failure metrics to a monitoring
		// system instead.
		// (https://github.com/italia/developers-italia-api/issues/73)
		log.Printf("error while dispatching webhook %s: %s", url, err.Error())

		return
	}

	// Drain and close so the connection can return to the pool, regardless
	// of whether the response is 2xx or an error status below.
	defer func() {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		//nolint:godox // need to implement this in the future
		// TODO: Replace this and send anonymous failure metrics to a monitoring
		// system instead.
		// (https://github.com/italia/developers-italia-api/issues/73)
		log.Printf("error while dispatching webhook %s: got HTTP %d", url, response.StatusCode)

		return
	}
}
