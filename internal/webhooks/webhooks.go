package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/gorm"
)

func DispatchWebhooks(event models.Event, gorm *gorm.DB) error {
	var webhooks []models.Webhook

	subject := fmt.Sprintf("/%s", event.EntityType)
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

	signature := ""

	for _, webhook := range webhooks {
		if webhook.Secret != "" && signature == "" {
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
	client := http.DefaultClient

	req, err := http.NewRequestWithContext(
		context.Background(),
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

	response, err := client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()
}
