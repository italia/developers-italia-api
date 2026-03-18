package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhooksEndpoints(t *testing.T) {
	tests := []TestCase{
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
		// TODO: enforce this?
		// {
		// 	query:    "PATCH /v1/webhooks with no Content-Type",
		// 	body:     "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },

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
