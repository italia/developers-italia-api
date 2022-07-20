package webhooks

import (
	"bytes"
	"context"
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
	stmt := gorm.Debug().
		Where(
			"entity_type = ? AND (entity_id = '' OR entity_id = ?)",
			event.EntityType,
			event.EntityID,
		)

	if err := stmt.Select("url").Find(&webhooks).Error; err != nil {
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
		go post(webhook.URL, jsonBody)
	}

	return nil
}

func post(url string, body []byte) {
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

	response, err := client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()
}
