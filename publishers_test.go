package main

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishersEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /publishers
		{
			description:         "GET the first page on publishers",
			query:               "GET /v1/publishers",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 25, len(data))

				// Default pagination size is 25, so there's another page and
				// next cursor should be present
				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd")

				firstPub := data[0]
				assert.NotEmpty(t, firstPub["email"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Equal(t, 2, len(firstPub["codeHosting"].([]interface{})))

				assertUUID(t, firstPub["id"])
				assertTimestamps(t, firstPub)
				assertOnlyKeys(t, firstPub, "id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active")
			},
		},
		{
			description:         "GET all the publishers, except the non active ones",
			query:               "GET /v1/publishers?page[size]=100",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 27, len(data))

				assertPaginationLinks(t, response, nil, nil)

				firstPub := data[0]
				assert.NotEmpty(t, firstPub["codeHosting"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Greater(t, len(firstPub["codeHosting"].([]interface{})), 0)

				assertUUID(t, firstPub["id"])
				assertTimestamps(t, firstPub)
				assertOnlyKeys(t, firstPub, "id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active")
			},
		},
		{
			description: "GET all publishers, including non active",
			query:       "GET /v1/publishers?all=true&page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 28, len(data))

				assertPaginationLinks(t, response, nil, nil)

				firstPub := data[0]
				assert.NotEmpty(t, firstPub["codeHosting"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Greater(t, len(firstPub["codeHosting"].([]interface{})), 0)

				assertUUID(t, firstPub["id"])
				assertTimestamps(t, firstPub)
				assertOnlyKeys(t, firstPub, "id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active")
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/publishers?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 2, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE4LTA1LTE2VDAwOjAwOjAwWiIsIjQ3ODA3ZTBjLTA2MTMtNGFlYS05OTE3LTU0NTVjYzZlZGRhZCJd")
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/publishers?page[size]=NOT_AN_INT",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		// TODO
		// {
		// 	description: "GET with page[size] bigger than the max of 100",
		// 	query:    "GET /v1/publishers?page[size]=200",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/publishers?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 2, len(data))

				assertPaginationLinks(t, response, "?page[before]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjkxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd", nil)
			},
		},
		{
			description: `GET with invalid "page[after]" query param`,
			query:       "GET /v1/publishers?page[after]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: "GET with page[before] query param",
			query:       "GET /v1/publishers?page[before]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjkxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 25, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd")
			},
		},
		{
			description: `GET with invalid "page[before]" query param`,
			query:       "GET /v1/publishers?page[before]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: `GET with "from" query param`,
			query:       "GET /v1/publishers?from=2018-11-10T00:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 14, len(data))
			},
		},
		{
			description: `GET with invalid "from" query param`,
			query:       "GET /v1/publishers?from=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description: `GET with "to" query param`,
			query:       "GET /v1/publishers?to=2018-11-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 13, len(data))
			},
		},
		{
			description: `GET with invalid "to" query param`,
			query:       "GET /v1/publishers?to=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},

		// GET /publishers/:id
		{
			description:         "Non-existent publisher",
			query:               "GET /v1/publishers/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:               "GET /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["codeHosting"])

				assert.IsType(t, []interface{}{}, response["codeHosting"])
				assert.Greater(t, len(response["codeHosting"].([]interface{})), 0)

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active")
			},
		},
		{
			description:         "GET publisher with alternativeId",
			query:               "GET /v1/publishers/alternative-id-12345",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "15fda7c4-6bbf-4387-8f89-258c1e6facb0", response["id"])
				assert.Equal(t, "alternative-id-12345", response["alternativeId"])

				assertTimestamps(t, response)
				assertOnlyKeys(t, response, "id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active", "alternativeId")
			},
		},

		// POST /publishers
		{
			query: "POST /v1/publishers",
			body:  `{"description": "new description", "codeHosting": [{"url" : "https://www.example-testcase-1.com"}], "email":"example-testcase-1@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://example-testcase-1.com", firstCodeHosting["url"])
				assert.Equal(t, true, firstCodeHosting["group"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)

				assert.Equal(t, true, response["active"])
				assert.Equal(t, "example-testcase-1@example.com", response["email"])

			},
		},
		{
			description: "POST publisher with alternativeId",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId":"12345", "description":"new description", "codeHosting": [{"url" : "https://www.example-testcase-2.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])
				assert.Equal(t, 1, len(response["codeHosting"].([]interface{})))

				// TODO: check codeHosting content
				assert.NotEmpty(t, response["codeHosting"])

				assert.Equal(t, "12345", response["alternativeId"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
			},
		},
		{
			description: "POST publisher with duplicate alternativeId",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId": "alternative-id-12345", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}], "email":"example-testcase-3-pass@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"alternativeId already exists","status":409}`,
		},
		{
			description: "POST publisher with alternativeId matching an existing id",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId": "2ded32eb-c45e-4167-9166-a44e18b8adde", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"Publisher with id '2ded32eb-c45e-4167-9166-a44e18b8adde' already exists","status":409}`,
		},
		{
			description: "POST publisher with empty alternativeId",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId": "", "description":"new description", "codeHosting": [{"url" : "https://gitlab.example.com/repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: alternativeId does not meet its size limits (too short)","status":422,"validationErrors":[{"field":"alternativeId","rule":"min","value":""}]}`,
		},
		{
			query: "POST /v1/publishers - NOT normalized URL validation passed",
			body:  `{"description":"new description", "codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://example-testcase-3.com", firstCodeHosting["url"])

				assertUUID(t, response["id"])
				assertTimestamps(t, response)
			},
		},
		{
			description: "POST publishers with duplicate URL (when normalized)",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://1-a.exAMple.org/code/repo"}], "description":"new description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"codeHosting.url already exists","status":409}`,
		},
		{
			description: "POST new publisher with an existing email",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://new-url.example.com"}], "email":"foobar@1.example.org", "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "foobar@1.example.org", response["email"])
			},
		},
		{
			description: "POST new publisher with an existing email (not normalized)",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://new-url.example.com"}], "email":"FoobaR@1.example.org", "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "foobar@1.example.org", response["email"])
			},
		},
		{
			description: "POST new publisher with no email",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://new-url.example.com"}], "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new publisher description", response["description"])

				email, exists := response["email"]
				assert.False(t, exists, "email key is present: %s", email)
			},
		},
		{
			description: "POST new publisher with empty email",
			query:       "POST /v1/publishers",
			body:        `{"email": "", "codeHosting": [{"url" : "https://new-url.example.com"}], "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: email is not a valid email","status":422,"validationErrors":[{"field":"email","rule":"email","value":""}]}`,
		},
		{
			query: "POST /v1/publishers - Description already exist",
			body:  `{"codeHosting": [{"url" : "https://example-testcase-xx3.com"}], "description": "Publisher description 1"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"description already exists","status":409}`,
		},
		{
			description: "POST new publisher with no description",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: description is required","status":422,"validationErrors":[{"field":"description","rule":"required","value":""}]}`,
		},
		{
			description: "POST new publisher with empty description",
			query:       "POST /v1/publishers",
			body:        `{"description":"", "codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: description is required","status":422,"validationErrors":[{"field":"description","rule":"required","value":""}]}`,
		},
		{
			description: "POST publisher with duplicate alternativeId",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId": "alternative-id-12345", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"alternativeId already exists","status":409}`,
		},
		{
			description: "POST publishers with invalid payload",
			query:       "POST /v1/publishers",
			body:        `{"description": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: codeHosting is required","status":422,"validationErrors":[{"field":"codeHosting","rule":"required","value":""}]}`,
		},
		{
			description: "POST publishers - wrong token",
			query:       "POST /v1/publishers",
			body:        `{"description":"new description", "codeHosting": [{"url" : "https://www.example-5.com"}]}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/publishers with invalid JSON",
			body:  `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "POST publishers with optional boolean field set to false",
			query:       "POST /v1/publishers",
			body:        `{"active": false, "description": "new description", "codeHosting": [{"url" : "https://www.example.com"}]}`,
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
			description: "POST publishers with codeHosting optional boolean field (group) set to false",
			query:       "POST /v1/publishers",
			body:        `{"description":"new description", "codeHosting": [{"url" : "https://www.example.com", "group": false}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://example.com", firstCodeHosting["url"])
				assert.Equal(t, false, firstCodeHosting["group"])
			},
		},
		{
			description: "POST publishers with validation errors",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "a"}], "email":"b"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid format: url is invalid, description is required, email is not a valid email", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 3, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST publishers with empty body",
			query:       "POST /v1/publishers",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},

		// PATCH /publishers/:id
		{
			description: "PATCH non existing publisher",
			query:       "PATCH /v1/publishers/NO_SUCH_PUBLISHER",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH a publisher",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new PATCHed description", "codeHosting": [{"url": "https://gitlab.example.org/patched-repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new PATCHed description", response["description"])
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})

				assert.Equal(t, "https://gitlab.example.org/patched-repo", firstCodeHosting["url"])
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH publishers with no codeHosting (should leave current codeHosting untouched)",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", response["id"])
				assert.Equal(t, "new description", response["description"])
				assert.Equal(t, "foobar@1.example.org", response["email"])

				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 2, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://1-a.example.org/code/repo", firstCodeHosting["url"])
				secondCodeHosting := codeHosting[1].(map[string]interface{})
				assert.Equal(t, "https://1-b.example.org/code/repo", secondCodeHosting["url"])

				assert.Equal(t, "2018-05-01T00:00:00Z", response["createdAt"])
				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH publishers with empty codeHosting",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description", "codeHosting": []}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"invalid format: codeHosting does not meet its size limits (too few items)","status":422,"validationErrors":[{"field":"codeHosting","rule":"gt","value":""}]}`,
		},
		{
			description: "PATCH a publisher via alternativeId",
			query:       "PATCH /v1/publishers/alternative-id-12345",
			body:        `{"description": "new PATCHed description via alternativeId", "codeHosting": [{"url": "https://gitlab.example.org/patched-repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new PATCHed description via alternativeId", response["description"])
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})

				assert.Equal(t, "https://gitlab.example.org/patched-repo", firstCodeHosting["url"])
				assert.Equal(t, "15fda7c4-6bbf-4387-8f89-258c1e6facb0", response["id"])

				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a publisher with alternativeId matching an existing id",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"alternativeId": "47807e0c-0613-4aea-9917-5455cc6eddad"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher with id '47807e0c-0613-4aea-9917-5455cc6eddad' already exists","status":409}`,
		},
		{
			description: "PATCH a publisher with duplicate description",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "Publisher description 2"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"description already exists","status":409}`,
		},
		{
			description: "PATCH publishers - wrong token",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
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
			description: "PATCH publisher with invalid JSON",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Publisher`, response["title"])
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "PATCH publishers with JSON with extra fields",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"unknown field in JSON input","status":422}`,
		},
		{
			description: "PATCH publisher with validation errors",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description", "codeHosting": [{"url": "INVALID_URL"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"invalid format: url is invalid","status":422,"validationErrors":[{"field":"url","rule":"url","value":"INVALID_URL"}]}`,
		},
		{
			description: "PATCH publishers with empty body",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"invalid or malformed JSON","status":400}`,
		},
		// TODO: enforce this?
		// {
		// 	query: "PATCH /v1/publishers with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }


		// JSON Patch
		{
			description: "PATCH a publisher with JSON Patch - replace description",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `[{"op": "replace", "path": "/description", "value": "new description via JSON Patch"}]`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new description via JSON Patch", response["description"])
				assert.Equal(t, "foobar@1.example.org", response["email"])
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", response["id"])
				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 2, len(codeHosting))
				created := assertRFC3339(t, response["createdAt"])
				updated := assertRFC3339(t, response["updatedAt"])
				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a publisher with JSON Patch - add codeHosting",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `[{"op": "add", "path": "/codeHosting/-", "value": {"url": "https://new-code-host.example.org", "group": false}}]`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "Publisher description 1", response["description"])
				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 3, len(codeHosting))
			},
		},
		{
			description: "PATCH a publisher with JSON Patch - malformed patch",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "this is not a JSON Patch"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json-patch+json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "can't update Publisher", response["title"])
				assert.Equal(t, "malformed JSON Patch", response["detail"])
			},
		},
		// DELETE /publishers/:id
		{
			description: "Delete non-existent publishers",
			query:       "DELETE /v1/publishers/eea19c82-0449-11ed-bd84-d8bbc146d165",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't delete Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE publishers with bad authentication",
			query:       "DELETE /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "DELETE /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},
		{
			description: "DELETE publisher via alternativeId",
			query:       "DELETE /v1/publishers/alternative-id-12345",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},

		// WebHooks

		// GET /publishers/:id/webhooks
		{
			query: "GET /v1/publishers/47807e0c-0613-4aea-9917-5455cc6eddad/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				// Default pagination size is 25, so all this publishers's logs fit into a page
				// and cursors should be empty
				assertPaginationLinks(t, response, nil, nil)

				firstWebhook := data[0]
				assert.Equal(t, "https://6-b.example.org/receiver", firstWebhook["url"])
				assert.Equal(t, "1702cd06-fffb-4d20-8f55-73e2a00ee052", firstWebhook["id"])
				assert.Equal(t, "2018-07-15T00:00:00Z", firstWebhook["createdAt"])
				assert.Equal(t, "2018-07-15T00:00:00Z", firstWebhook["updatedAt"])

				assertOnlyKeys(t, firstWebhook, "id", "url", "createdAt", "updatedAt")
			},
		},
		{
			description: "GET webhooks for non existing publisher",
			query:       "GET /v1/publishers/NO_SUCH_publishers/webhooks",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			description: "GET webhooks for publisher without webhooks",
			query:       "GET /v1/publishers/b97446f8-fe06-472c-9b26-c40150cac77f/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/publishers/d6ddc11a-ff85-4f0f-bb87-df38b2a9b394/webhooks?page[size]=1",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDE4LTA3LTE1VDAwOjAwOjAwWiIsIjhmMzczYThjLTFmNTUtNDVlNC04NTQ5LTA1Y2Q2MzJhMmFkZCJd")
			},
		},

		// POST /publishers/:id/webhooks
		{
			description: "POST webhook for non existing publisher",
			query:       "POST /v1/publishers/NO_SUCH_publishers/webhooks",
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
			query: "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
			description: "POST webhook with non-normalized URL",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:        `{"url": "https://www.new.example.org/"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://new.example.org", response["url"])
			},
		},
		{
			description: "POST publishers webhook - wrong token",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
			description: "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks with JSON with extra fields",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
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
		// 	query: "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },
	}

	runTestCases(t, tests)
}

func TestPublishersPatchDBChecks(t *testing.T) {
	t.Run("PATCH publisher persists changes to DB", func(t *testing.T) {
		loadFixtures(t)

		const publisherID = "2ded32eb-c45e-4167-9166-a44e18b8adde"

		body := `{"description": "new PATCHed description", "codeHosting": [{"url": "https://gitlab.example.org/patched-repo"}]}`
		req, err := http.NewRequest("PATCH", "/v1/publishers/"+publisherID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		assert.Equal(t, "new PATCHed description", dbValue(t, "publishers", "description", "id", publisherID))

		assert.Equal(t, 0, dbCount(t, "publishers_code_hosting", "url", "https://1-a.example.org/code/repo"))
		assert.Equal(t, 0, dbCount(t, "publishers_code_hosting", "url", "https://1-b.example.org/code/repo"))

		assert.Equal(t, publisherID, dbValue(t, "publishers_code_hosting", "publisher_id", "url", "https://gitlab.example.org/patched-repo"))
		assert.Equal(t, 1, dbCount(t, "publishers_code_hosting", "publisher_id", publisherID))
	})
}

func TestPublishersDeleteDBChecks(t *testing.T) {
	t.Run("DELETE publisher removes code hosting rows", func(t *testing.T) {
		// TODO: make this pass
		t.Skip("publishers_code_hosting rows are not deleted on publisher DELETE (implementation bug)")

		loadFixtures(t)

		const publisherID = "15fda7c4-6bbf-4387-8f89-258c1e6fafb1"

		req, err := http.NewRequest("DELETE", "/v1/publishers/"+publisherID, nil)
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

		assert.Equal(t, 0, dbCount(t, "publishers_code_hosting", "publisher_id", publisherID))
	})
}
