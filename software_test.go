package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /software
		{
			description: "GET the first page on software",
			query:       "GET /v1/software",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 25, len(data))

				// Default pagination size is 25, so there's another page and
				// next cursor should be present
				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd")

				firstSoftware := data[0]
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				assertUUID(t, firstSoftware["id"])
				assertTimestamps(t, firstSoftware)

				assert.Equal(t, true, firstSoftware["active"])

				assertOnlyKeys(t, firstSoftware, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},
		{
			description: "GET all the software, except the non active ones",
			query:       "GET /v1/software?page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 30, len(data))

				assertPaginationLinks(t, response, nil, nil)

				firstSoftware := data[0]
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				assertUUID(t, firstSoftware["id"])

				assert.Equal(t, true, firstSoftware["active"])

				assertTimestamps(t, firstSoftware)
				assertOnlyKeys(t, firstSoftware, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},
		{
			description: "GET all software, including non active",
			query:       "GET /v1/software?all=true&page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 31, len(data))

				assertPaginationLinks(t, response, nil, nil)

				firstSoftware := data[0]
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				assertUUID(t, firstSoftware["id"])
				assertTimestamps(t, firstSoftware)
				assertOnlyKeys(t, firstSoftware, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},
		{
			description: "GET software with a specific URL",
			query:       "GET /v1/software?url=https://1-a.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				assertPaginationLinks(t, response, nil, nil)

				firstSoftware := data[0]
				assert.Equal(t, "-", firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				assert.Equal(t, "c353756e-8597-4e46-a99b-7da2e141603b", firstSoftware["id"])

				assert.Equal(t, "2014-05-01T00:00:00Z", firstSoftware["createdAt"])
				assert.Equal(t, "2014-05-01T00:00:00Z", firstSoftware["updatedAt"])

				assertOnlyKeys(t, firstSoftware, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},
		{
			description: "GET software with a non-normalized URL filter",
			query:       "GET /v1/software?url=https://www.1-a.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				firstSoftware := data[0]
				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])
			},
		},
		{
			description: "GET software with a specific URL that doesn't exist",
			query:       "GET /v1/software?url=https://no.such.url.in.db.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))

				assertPaginationLinks(t, response, nil, nil)
			},
		},
		{
			description: "GET with url filter excludes inactive software",
			query:       "GET /v1/software?url=https://31-a.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 2, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE0LTA1LTE2VDAwOjAwOjAwWiIsIjlmMTM1MjY4LWEzN2UtNGVhZC05NmVjLWU0YTI0YmI5MzQ0YSJd")
			},
		},
		{
			description: "GET with invalid format for page[size] query param",
			query:       "GET /v1/software?page[size]=NOT_AN_INT",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "page[size] must be an integer", response["detail"])
			},
		},
		{
			description: "GET with page[size] bigger than the max of 100 caps the size",
			query:       "GET /v1/software?page[size]=200",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				items := assertListResponse(t, response)
				assert.LessOrEqual(t, len(items), 100)
			},
		},
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/software?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 5, len(data))

				assertPaginationLinks(t, response, "?page[before]=WyIyMDE1LTA1LTExVDAwOjAwOjAwWiIsIjgzZTdhMzVlLTMyOGItNDg5MS1iNjBiLTU5NzkyZTAxYzU5ZSJd", nil)
			},
		},
		{
			description: `GET with invalid "page[after]" query param`,
			query:       "GET /v1/software?page[after]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: "GET with page[before] query param",
			query:       "GET /v1/software?page[before]=WyIyMDE1LTA1LTExVDAwOjAwOjAwWiIsIjgzZTdhMzVlLTMyOGItNDg5MS1iNjBiLTU5NzkyZTAxYzU5ZSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 25, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd")
			},
		},
		{
			description: `GET with invalid "page[before]" query param`,
			query:       "GET /v1/software?page[before]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: `GET with "from" query param`,
			query:       "GET /v1/software?from=2015-04-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 7, len(data))
			},
		},
		{
			description: `GET with invalid "from" query param`,
			query:       "GET /v1/software?from=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description: `GET with "to" query param`,
			query:       "GET /v1/software?to=2014-11-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 13, len(data))
			},
		},
		{
			description: `GET with invalid "to" query param`,
			query:       "GET /v1/software?to=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},

		// GET /software/:id
		{
			description:         "Non-existent software",
			query:               "GET /v1/software/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:               "GET /v1/software/e7576e7f-9dcf-4979-b9e9-d8cdcad3b60e",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.Equal(t, "https://8-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Equal(t, 1, len(response["aliases"].([]interface{})))

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},
		{
			description:         "GET software with vitality field",
			query:               "GET /v1/software/9f135268-a37e-4ead-96ec-e4a24bb9344a",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.Equal(t, "https://2-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Equal(t, 1, len(response["aliases"].([]interface{})))

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")
			},
		},

		// POST /software
		{
			query: "POST /v1/software",
			body:  `{"publiccodeYml": "-", "url": "https://software.example.org"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.org", response["url"])
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Empty(t, response["aliases"].([]interface{}))

				assertUUID(t, response["id"])
				assertTimestamps(t, response)

				assert.Equal(t, true, response["active"])

				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")

			},
		},
		{
			description: "POST software with aliases",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "url": "https://software.example.org", "aliases": ["https://software-1.example.org", "https://software-2.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.org", response["url"])
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://software-1.example.org", aliases[0])
				assert.Equal(t, "https://software-2.example.org", aliases[1])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")

			},
		},
		{
			description: "POST software with non-normalized URL",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "url": "https://www.software.example.org", "aliases": ["https://www.alias.example.org/"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.org", response["url"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))
				assert.Equal(t, "https://alias.example.org", aliases[0])
			},
		},
		{
			description: "POST software with vitality field",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "url": "https://software.example.net", "vitality": "90,90,90"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			expectedBody:        "x",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.net", response["url"])
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Empty(t, response["aliases"].([]interface{}))

				assert.Equal(t, "90,90,90", response["vitality"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active", "vitality", "catalogId")

			},
		},
		{
			description: "POST software with analysis field is rejected",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "url": "https://analysis.example.org", "analysis": {"badges": {"v": 1, "score": 90}}}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't create Software", response["title"])
			},
		},
		{
			description: "POST software with invalid payload",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Software","detail":"invalid format: url is required","status":422,"validationErrors":[{"field":"url","rule":"required","value":""}]}`,
		},
		{
			description: "POST software - wrong token",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml":  "-", "url": "https://software.example.org"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/software with invalid JSON",
			body:  `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "POST /v1/software with JSON with extra fields",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Software","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "POST software with optional boolean field set to false",
			query:       "POST /v1/software",
			body:        `{"active": false, "url": "https://example.org", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, false, response["active"])
			},
		},
		{
			description: "POST software with validation errors",
			query:       "POST /v1/software",
			body:        `{"url":"", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid format: url is required", response["detail"])

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
			description: "POST software with empty body",
			query:       "POST /v1/software",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }

		// PATCH /software/:id
		{
			description: "PATCH non-existing software",
			query:       "PATCH /v1/software/NO_SUCH_SOFTWARE",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't update Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH a software resource",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org", "aliases": ["https://software.example.com", "https://software-old.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://software-old.example.org", aliases[0])
				assert.Equal(t, "https://software.example.com", aliases[1])

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software with no aliases (should leave current aliases untouched)",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			expectedBody:        "",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software with empty aliases (should remove aliases)",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org", "aliases": []}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 0, len(aliases))

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software with an already existing URL (of another software)",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://21-b.example.org/code/repo"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"url already exists","status":409}`,
		},
		{
			description: "PATCH software, change active",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"active": false}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, false, response["active"])
				assert.Equal(t, "https://18-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])

				assert.Equal(t, "-", response["publiccodeYml"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software, switch url and alias",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"url": "https://18-b.example.org/code/repo", "aliases": ["https://18-a.example.org/code/repo"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://18-b.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-a.example.org/code/repo", aliases[0])

				assert.Equal(t, "-", response["publiccodeYml"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software, vitality",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"vitality": "80,90,99"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://18-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])

				assert.Equal(t, "-", response["publiccodeYml"])
				assert.Equal(t, "80,90,99", response["vitality"])

				assertUUID(t, response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a software resource with JSON Patch - replace",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `[{"op": "replace", "path": "/publiccodeYml", "value": "new publiccode data"}]`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://18-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])

				assert.Equal(t, "new publiccode data", response["publiccodeYml"])
				assert.Equal(t, "59803fb7-8eec-4fe5-a354-8926009c364a", response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a software resource with JSON Patch - replace vitality",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `[{"op": "replace", "path": "/vitality", "value": "10,11"}]`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://18-a.example.org/code/repo", response["url"])
				assert.Equal(t, "10,11", response["vitality"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 1, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])

				assert.Equal(t, "-", response["publiccodeYml"])
				assert.Equal(t, "59803fb7-8eec-4fe5-a354-8926009c364a", response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},

		{
			description: "PATCH a software resource with JSON Patch - add",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `[{"op": "add", "path": "/aliases/-", "value": "https://18-c.example.org"}]`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, true, response["active"])
				assert.Equal(t, "https://18-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[0])
				assert.Equal(t, "https://18-c.example.org", aliases[1])

				assert.Equal(t, "-", response["publiccodeYml"])
				assert.Equal(t, "59803fb7-8eec-4fe5-a354-8926009c364a", response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a software resource with JSON Patch as Content-Type, but non JSON Patch payload",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},

			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "malformed JSON Patch", response["detail"])
			},
		},
		{
			description: "PATCH software using an already taken URL as url",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"url": "https://15-b.example.org/code/repo"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"url already exists","status":409}`,
		},
		{
			description: "PATCH software using an already taken URL as an alias",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"aliases": ["https://16-b.example.org/code/repo"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"url already exists","status":409}`,
		},
		{
			description: "PATCH software - wrong token",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH software with invalid JSON",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "PATCH software with JSON with extra fields",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "-", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "PATCH software with validation errors",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"url": "INVALID_URL", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
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
			description: "PATCH software with an empty url",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"url": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"invalid format: url is invalid","status":422,"validationErrors":[{"field":"url","rule":"url","value":""}]}`,
		},
		{
			description: "PATCH software with empty body",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }

		// DELETE /software/:id
		{
			description: "Delete non-existent software",
			query:       "DELETE /v1/software/eea19c82-0449-11ed-bd84-d8bbc146d165",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't delete Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE software with bad authentication",
			query:       "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},

		// GET /software/:id/logs
		{
			query: "GET /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 3, len(data))

				// Default pagination size is 25, so all this software's logs fit into a page
				// and cursors should be empty
				assertPaginationLinks(t, response, nil, nil)

				var prevCreatedAt *time.Time = nil
				for _, item := range data {
					log := item

					assert.NotEmpty(t, log["message"])

					assertUUID(t, log["id"])

					createdAt := assertRFC3339(t, log["createdAt"])
					assertRFC3339(t, log["updatedAt"])

					assert.Equal(t, "/software/c353756e-8597-4e46-a99b-7da2e141603b", log["entity"])

					assertOnlyKeys(t, log, "id", "createdAt", "updatedAt", "message", "entity")

					// Check the logs are ordered by descending createdAt
					if prevCreatedAt != nil {
						assert.GreaterOrEqual(t, *prevCreatedAt, createdAt)
					}

					prevCreatedAt = &createdAt
				}
			},
		},
		{
			description: "GET logs for non existing software",
			query:       "GET /v1/software/NO_SUCH_SOFTWARE/logs",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "Software was not found", response["detail"])
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 2, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDEwLTAxLTE1VDIzOjU5OjU5WiIsIjEyZjMwZDllLTA0MmUtMTFlZC04ZGRjLWQ4YmJjMTQ2ZDE2NSJd")
			},
		},

		// POST /software/:id/logs
		{
			description: "POST logs for non existing software",
			query:       "POST /v1/software/NO_SUCH_SOFTWARE/logs",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "Software was not found", response["detail"])
			},
		},
		{
			query: "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:  `{"message": "New software log from test suite"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "New software log from test suite", response["message"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)

				assert.Equal(t, "/software/c353756e-8597-4e46-a99b-7da2e141603b", response["entity"])

			},
		},
		{
			description: "POST software log - wrong token",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `{"message": "new log"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST log with invalid JSON",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs with JSON with extra fields",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `{"message": "new log", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Log","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "POST log with validation errors",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `{"message": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid format: message is required", response["detail"])

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
			description: "POST log with empty body",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/logs with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },

		// GET /software/:id/webhooks
		{
			query: "GET /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				// Default pagination size is 25, so all this software's logs fit into a page
				// and cursors should be empty
				assertPaginationLinks(t, response, nil, nil)

				firstWebhook := data[0]
				assert.Equal(t, "https://1-b.example.org/receiver", firstWebhook["url"])
				assert.Equal(t, "007bc84a-7e2d-43a0-b7e1-a256d4114aa7", firstWebhook["id"])
				assert.Equal(t, "2017-05-01T00:00:00Z", firstWebhook["createdAt"])
				assert.Equal(t, "2017-05-01T00:00:00Z", firstWebhook["updatedAt"])

				assertOnlyKeys(t, firstWebhook, "id", "url", "createdAt", "updatedAt")
			},
		},
		{
			description: "GET webhooks for non existing software",
			query:       "GET /v1/software/NO_SUCH_SOFTWARE/webhooks",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			description: "GET webhooks for software without webhooks",
			query:       "GET /v1/software/e7576e7f-9dcf-4979-b9e9-d8cdcad3b60e/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software/9f135268-a37e-4ead-96ec-e4a24bb9344a/webhooks?page[size]=1",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE3LTA1LTAxVDAwOjAwOjAwWiIsImQ2MzM0MDAwLTY5YTgtNDNhMS1hYjQzLTUwYmIwNGUxNGVlZCJd")
			},
		},

		// POST /software/:id/webhooks
		{
			description: "POST webhooks for non existing software",
			query:       "POST /v1/software/NO_SUCH_SOFTWARE/webhooks",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			query: "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:  `{"url": "https://new.example.org", "secret": "xyz"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://new.example.org", response["url"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "url", "createdAt", "updatedAt")

			},
		},
		{
			description: "POST software webhook - wrong token",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `{"url": "https://new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST webhook with invalid JSON",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks with JSON with extra fields",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `{"url": "https://new.example.org", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Webhook","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "POST webhook with validation errors",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `{"url": "INVALID_URL"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
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
			description: "POST webhook with empty body",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },
	}

	runTestCases(t, tests)
}

func TestSoftwarePostDBChecks(t *testing.T) {
	t.Run("POST software persists changes to DB", func(t *testing.T) {
		loadFixtures(t)

		body := `{"publiccodeYml":"persisted-publiccode","url":"https://www.software-dbcheck.example.org/","aliases":["https://www.alias-one.example.org/","https://alias-two.example.org"]}`
		req, err := newTestRequest("POST", "/v1/software", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		softwareID := dbValue(t, "software_urls", "software_id", "url", "https://software-dbcheck.example.org")
		assert.Equal(t, "persisted-publiccode", dbValue(t, "software", "publiccode_yml", "id", softwareID))
		assert.Equal(t, 3, dbCount(t, "software_urls", "software_id", softwareID))
	})

	t.Run("POST software does not accept analysis field", func(t *testing.T) {
		loadFixtures(t)

		body := `{"publiccodeYml": "-", "url": "https://analysis-db.example.org", "analysis": {"badges": {"v": 1, "score": 90}}}`
		req, err := newTestRequest("POST", "/v1/software", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 422, res.StatusCode)
	})
}

func TestSoftwarePatchDBChecks(t *testing.T) {
	t.Run("PATCH software persists changes to DB", func(t *testing.T) {
		loadFixtures(t)

		const softwareID = "59803fb7-8eec-4fe5-a354-8926009c364a"

		body := `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org", "aliases": ["https://software.example.com", "https://software-old.example.org"]}`
		req, err := newTestRequest("PATCH", "/v1/software/"+softwareID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		assert.Equal(t, "publiccodedata", dbValue(t, "software", "publiccode_yml", "id", softwareID))

		assert.Equal(t, 0, dbCount(t, "software_urls", "url", "https://18-a.example.org/code/repo"))
		assert.Equal(t, 0, dbCount(t, "software_urls", "url", "https://18-b.example.org/code/repo"))

		assert.Equal(t, softwareID, dbValue(t, "software_urls", "software_id", "url", "https://software-new.example.org"))
		assert.Equal(t, 3, dbCount(t, "software_urls", "software_id", softwareID))
	})

	t.Run("PATCH software does not accept analysis field", func(t *testing.T) {
		loadFixtures(t)

		const softwareID = "59803fb7-8eec-4fe5-a354-8926009c364a"

		body := `{"analysis": {"badges": {"v": 1, "score": 75}}}`
		req, err := newTestRequest("PATCH", "/v1/software/"+softwareID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 422, res.StatusCode)
	})
}

func TestSoftwareAnalysisEndpoints(t *testing.T) {
	const softwareID = "59803fb7-8eec-4fe5-a354-8926009c364a"
	const missingID = "00000000-0000-0000-0000-000000000000"

	tests := []TestCase{
		{
			description:         "GET analysis on software with no analysis returns empty object",
			query:               "GET /v1/software/" + softwareID + "/analysis",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Empty(t, response)
			},
		},
		{
			description: "PATCH analysis adds namespace with timestamp",
			query:       "PATCH /v1/software/" + softwareID + "/analysis",
			body:        `{"badges": {"v": 1, "score": 90}}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				badges := response["badges"].(map[string]interface{})

				assert.Equal(t, float64(1), badges["v"])
				assert.Equal(t, float64(90), badges["score"])
				assertRFC3339(t, badges["t"])
			},
		},
		{
			description: "PATCH analysis missing v field returns 422",
			query:       "PATCH /v1/software/" + softwareID + "/analysis",
			body:        `{"badges": {"score": 90}}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't update Software analysis", response["title"])
			},
		},
		{
			description:         "GET analysis on nonexistent software returns 404",
			query:               "GET /v1/software/" + missingID + "/analysis",
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't get Software analysis", response["title"])
			},
		},
		{
			description: "PATCH analysis on nonexistent software returns 404",
			query:       "PATCH /v1/software/" + missingID + "/analysis",
			body:        `{"badges": {"v": 1}}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't update Software analysis", response["title"])
			},
		},
	}

	runTestCases(t, tests)
}

func TestSoftwareAnalysisDBChecks(t *testing.T) {
	t.Run("PATCH analysis persists to DB", func(t *testing.T) {
		loadFixtures(t)

		const softwareID = "59803fb7-8eec-4fe5-a354-8926009c364a"

		body := `{"badges": {"v": 1, "score": 75}}`
		req, err := newTestRequest("PATCH", "/v1/software/"+softwareID+"/analysis", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		raw := dbValue(t, "software", "analysis", "id", softwareID)

		var analysis map[string]interface{}
		require.NoError(t, json.NewDecoder(strings.NewReader(raw)).Decode(&analysis))

		badges := analysis["badges"].(map[string]interface{})
		assert.Equal(t, float64(1), badges["v"])
		assert.Equal(t, float64(75), badges["score"])
		assertRFC3339(t, badges["t"])
	})

}

func TestSoftwareDeleteDBChecks(t *testing.T) {
	t.Run("DELETE software removes software_urls rows", func(t *testing.T) {
		loadFixtures(t)

		const softwareID = "11e101c4-f989-4cc4-a665-63f9f34e83f6"

		req, err := newTestRequest("DELETE", "/v1/software/"+softwareID, nil)
		if err != nil {
			assert.Fail(t, err.Error())
		}
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		assert.Nil(t, err)
		assert.Equal(t, 204, res.StatusCode)

		assert.Equal(t, 0, dbCount(t, "software_urls", "software_id", softwareID))
	})
}
