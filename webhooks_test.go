package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhooksEndpoints(t *testing.T) {
	tests := []TestCase{
		// POST /software/webhooks
		{
			description: "POST webhook with too-short secret returns 422",
			query:       "POST /v1/software/webhooks",
			body:        `{"url": "https://example.org/receiver", "secret": "tooshort"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't create Webhook", response["title"])
			},
		},
		{
			description: "POST webhook with valid 16-char secret returns 201",
			query:       "POST /v1/software/webhooks",
			body:        `{"url": "https://example.org/receiver", "secret": "1234567890abcdef"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assertUUID(t, response["id"])
				assert.Equal(t, "https://example.org/receiver", response["url"])
			},
		},
		// GET /webhooks/:id
		{
			query:               "GET /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			expectedCode:        200,
			expectedBody:        `{"id":"007bc84a-7e2d-43a0-b7e1-a256d4114aa7","url":"https://1-b.example.org/receiver","createdAt":"2017-05-01T00:00:00Z","updatedAt":"2017-05-01T00:00:00Z"}`,
			expectedContentType: "application/json",
		},
		{
			description:  "Non-existent webhook",
			query:        "GET /v1/webhooks/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode: 404,
			expectedBody: `{"title":"can't get Webhook","detail":"Webhook was not found","status":404}`,

			expectedContentType: "application/problem+json",
		},

		// PATCH /webhooks/:id
		{
			query: "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:  `{"url": "https://new.example.org/receiver"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "007bc84a-7e2d-43a0-b7e1-a256d4114aa7", response["id"])
				assert.Equal(t, "https://new.example.org/receiver", response["url"])
				assert.Equal(t, "2017-05-01T00:00:00Z", response["createdAt"])

				assertRFC3339(t, response["updatedAt"])
				assertOnlyKeys(t, response, "id", "url", "createdAt", "updatedAt")
			},
		},
		{
			description: "PATCH webhook with non-normalized URL",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "https://www.new.example.org/receiver/"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://new.example.org/receiver", response["url"])
			},
		},
		{
			description: "PATCH webhook - wrong token",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "https://new.example.org/receiver"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH /v1/webhooks with invalid JSON",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7 with JSON with extra fields",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "https://new.example.org/receiver", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Webhook","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "PATCH webhook with validation errors",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "INVALID_URL"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid format: url is invalid", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "PATCH /v1/webhooks with empty body",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		// DELETE /webhooks/:id
		{
			description: "Delete non-existent webhook",
			query:       "DELETE /v1/webhooks/NO_SUCH_WEBHOOK",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode: 404,
			// This error is different from because it's returned directly from Fiber's
			// route constraints, so we don't need to hit the database to find the resource
			// because we already know that's not a valid webhook id looking at its format.
			expectedBody:        `{"title":"Not Found","detail":"Cannot DELETE /v1/webhooks/NO_SUCH_WEBHOOK","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE webhook with bad authentication",
			query:       "DELETE /v1/webhooks/1702cd06-fffb-4d20-8f55-73e2a00ee052",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "DELETE /v1/webhooks/24bc1b5d-fe81-47be-9d55-910f820bdd04",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},
	}

	runTestCases(t, tests)
}
func TestWebhookJSONPatch(t *testing.T) {
	const webhookID = "007bc84a-7e2d-43a0-b7e1-a256d4114aa7"

	t.Run("JSON Patch replaces the URL", func(t *testing.T) {
		loadFixtures(t)

		body := `[{"op":"replace","path":"/url","value":"https://1-b.example.org/json-patched"}]`

		req, err := newTestRequest("PATCH", "/v1/webhooks/"+webhookID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		assert.Equal(t, "https://1-b.example.org/json-patched", dbValue(t, "webhooks", "url", "id", webhookID))
		assert.Equal(t, 0, dbCount(t, "webhooks", "url", "https://1-b.example.org/receiver"))
	})

	t.Run("JSON Patch leaves the secret alone", func(t *testing.T) {
		loadFixtures(t)

		const secret = "1234567890abcdef"
		const webhookURL = "https://secret-keeper.example.org/receiver"

		createReq, err := newTestRequest("POST", "/v1/software/webhooks",
			strings.NewReader(`{"url":"`+webhookURL+`","secret":"`+secret+`"}`))
		require.NoError(t, err)
		createReq.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		createRes, err := app.Test(createReq, -1)
		require.NoError(t, err)
		require.Equal(t, 200, createRes.StatusCode)

		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(createRes.Body).Decode(&created))
		createdID := created["id"].(string)

		require.Equal(t, secret, dbValue(t, "webhooks", "secret", "id", createdID))

		body := `[{"op":"replace","path":"/url","value":"https://secret-keeper.example.org/patched"}]`

		req, err := newTestRequest("PATCH", "/v1/webhooks/"+createdID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		assert.Equal(t, secret, dbValue(t, "webhooks", "secret", "id", createdID))
		assert.Equal(t, "https://secret-keeper.example.org/patched", dbValue(t, "webhooks", "url", "id", createdID))
	})

	t.Run("JSON Patch rejects an empty URL", func(t *testing.T) {
		loadFixtures(t)

		body := `[{"op":"replace","path":"/url","value":""}]`

		req, err := newTestRequest("PATCH", "/v1/webhooks/"+webhookID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 422, res.StatusCode)

		assert.Equal(t, "https://1-b.example.org/receiver", dbValue(t, "webhooks", "url", "id", webhookID))
	})

	t.Run("JSON Patch rejects a path the webhook does not have", func(t *testing.T) {
		loadFixtures(t)

		body := `[{"op":"replace","path":"/secret","value":"1234567890abcdef"}]`

		req, err := newTestRequest("PATCH", "/v1/webhooks/"+webhookID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 422, res.StatusCode)
	})

	t.Run("JSON Patch rejects a malformed document", func(t *testing.T) {
		loadFixtures(t)

		body := `{"url":"https://1-b.example.org/not-a-patch"}`

		req, err := newTestRequest("PATCH", "/v1/webhooks/"+webhookID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 400, res.StatusCode)
	})

	t.Run("JSON Patch on a webhook that does not exist", func(t *testing.T) {
		loadFixtures(t)

		body := `[{"op":"replace","path":"/url","value":"https://1-b.example.org/json-patched"}]`

		req, err := newTestRequest("PATCH", "/v1/webhooks/00000000-0000-0000-0000-000000000000", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json-patch+json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 404, res.StatusCode)
	})
}
