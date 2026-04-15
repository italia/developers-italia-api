package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	italiaID = "a8e5e6d7-0b1c-4f2a-8e3d-9c4b5a6f7e8d"
	swissID  = "b9f6f7e8-1c2d-4f3b-9f4e-0d5c6b7a8f9e"

	// Software fixtures with known catalog assignments.
	italiaSoftwareID = "c353756e-8597-4e46-a99b-7da2e141603b" // catalog_id = italiaID
	swissSoftwareID  = "9f135268-a37e-4ead-96ec-e4a24bb9344a" // catalog_id = swissID
	rootSoftwareID   = "18348f13-1076-4a1e-b204-ed541b824d64" // catalog_id IS NULL (root)

	// Publisher fixtures with known catalog assignments.
	italiaPublisherID = "2ded32eb-c45e-4167-9166-a44e18b8adde" // catalog_id = italiaID
	swissPublisherID  = "47807e0c-0613-4aea-9917-5455cc6eddad" // catalog_id = swissID
	rootPublisherID   = "d6ddc11a-ff85-4f0f-bb87-df38b2a9b394" // catalog_id IS NULL (root)
)

func TestCatalogEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /catalogs
		{
			description:         "GET catalogs",
			query:               "GET /v1/catalogs",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 2, len(data))

				first := data[0]
				assertUUID(t, first["id"])
				assertTimestamps(t, first)
				assertOnlyKeys(t, first, "id", "name", "alternativeId", "active", "sources", "createdAt", "updatedAt")
			},
		},

		// GET /catalogs/:id
		{
			description:         "GET catalog by id",
			query:               "GET /v1/catalogs/" + italiaID,
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaID, response["id"])
				assert.Equal(t, "Italian Catalog", response["name"])
				assert.Equal(t, "italia", response["alternativeId"])
				assertOnlyKeys(t, response, "id", "name", "alternativeId", "active", "sources", "createdAt", "updatedAt")
			},
		},
		{
			description:         "GET catalog by alternativeId",
			query:               "GET /v1/catalogs/swiss",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, swissID, response["id"])
				assert.Equal(t, "Swiss Catalog", response["name"])
			},
		},
		{
			description:         "GET catalog not found",
			query:               "GET /v1/catalogs/nonexistent",
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't get Catalog","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "GET root catalog (∅) returns 404 — root is implicit",
			query:               "GET /v1/catalogs/%E2%88%85",
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't get Catalog","detail":"Catalog was not found","status":404}`,
		},

		// POST /catalogs
		{
			description: "POST catalog",
			query:       "POST /v1/catalogs",
			body:        `{"name": "New Catalog", "sources": [{"url": "https://github.com/example/new-catalog"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assertUUID(t, response["id"])
				assert.Equal(t, "New Catalog", response["name"])
				assertOnlyKeys(t, response, "id", "name", "active", "sources", "createdAt", "updatedAt")
			},
		},
		{
			description: "POST catalog with alternativeId",
			query:       "POST /v1/catalogs",
			body:        `{"name": "Another Catalog", "alternativeId": "another", "sources": [{"url": "https://github.com/example/another"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "another", response["alternativeId"])
			},
		},
		{
			description: "POST catalog duplicate alternativeId",
			query:       "POST /v1/catalogs",
			body:        `{"name": "Dup", "alternativeId": "italia", "sources": [{"url": "https://github.com/example/dup"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Catalog","detail":"already exists","status":409}`,
		},
		{
			description: "POST catalog missing name",
			query:       "POST /v1/catalogs",
			body:        `{}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Catalog","detail":"invalid format: name is required, sources is required","status":422,"validationErrors":[{"field":"name","rule":"required","value":""},{"field":"sources","rule":"required","value":""}]}`,
		},
		{
			description:         "POST catalog - no token",
			query:               "POST /v1/catalogs",
			body:                `{"name": "Unauth"}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// PATCH /catalogs/:id
		{
			description: "PATCH catalog",
			query:       "PATCH /v1/catalogs/" + italiaID,
			body:        `{"name": "Updated Italian Catalog"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaID, response["id"])
				assert.Equal(t, "Updated Italian Catalog", response["name"])
			},
		},
		{
			description: "PATCH catalog - root returns 404",
			query:       "PATCH /v1/catalogs/%E2%88%85",
			body:        `{"name": "x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Catalog","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "PATCH catalog - no token",
			query:               "PATCH /v1/catalogs/" + italiaID,
			body:                `{"name": "x"}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// DELETE /catalogs/:id
		{
			description: "DELETE catalog - 409 if has publishers",
			query:       "DELETE /v1/catalogs/" + italiaID,
			headers: map[string][]string{
				"Authorization": {goodToken},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't delete Catalog","detail":"Catalog still has associated publishers or software","status":409}`,
		},
		{
			description: "DELETE catalog - root returns 404",
			query:       "DELETE /v1/catalogs/%E2%88%85",
			headers: map[string][]string{
				"Authorization": {goodToken},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't delete Catalog","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "DELETE catalog - no token",
			query:               "DELETE /v1/catalogs/" + italiaID,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// POST /catalogs/:id/publishers
		{
			description: "POST catalog publisher",
			query:       "POST /v1/catalogs/" + italiaID + "/publishers",
			body:        `{"description":"New Publisher","codeHosting":[{"url":"https://example.org/code"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assertUUID(t, response["id"])
				assert.Equal(t, italiaID, response["catalogId"])
			},
		},
		{
			description: "POST catalog publisher - root catalog (∅)",
			query:       "POST /v1/catalogs/%E2%88%85/publishers",
			body:        `{"description":"Root Publisher","codeHosting":[{"url":"https://example.org/root-pub"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Nil(t, response["catalogId"])
			},
		},
		{
			description: "POST catalog publisher - catalog not found",
			query:       "POST /v1/catalogs/nonexistent/publishers",
			body:        `{"description":"x","codeHosting":[{"url":"https://example.org/x"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "POST catalog publisher - no token",
			query:               "POST /v1/catalogs/" + italiaID + "/publishers",
			body:                `{"description":"x","codeHosting":[{"url":"https://example.org/x"}]}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// PATCH /catalogs/:id/publishers/:publisherId
		{
			description: "PATCH catalog publisher",
			query:       "PATCH /v1/catalogs/" + italiaID + "/publishers/" + italiaPublisherID,
			body:        `{"description":"Updated Publisher"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaPublisherID, response["id"])
				assert.Equal(t, "Updated Publisher", response["description"])
			},
		},
		{
			description: "PATCH catalog publisher - wrong catalog returns 404",
			query:       "PATCH /v1/catalogs/" + swissID + "/publishers/" + italiaPublisherID,
			body:        `{"description":"x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher was not found","status":404}`,
		},
		{
			description: "PATCH catalog publisher - root catalog (∅)",
			query:       "PATCH /v1/catalogs/%E2%88%85/publishers/" + rootPublisherID,
			body:        `{"description":"Updated Root Publisher"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, rootPublisherID, response["id"])
			},
		},
		{
			description: "PATCH catalog publisher - catalog-scoped publisher rejected for root catalog",
			query:       "PATCH /v1/catalogs/%E2%88%85/publishers/" + italiaPublisherID,
			body:        `{"description":"x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher was not found","status":404}`,
		},
		{
			description:         "PATCH catalog publisher - no token",
			query:               "PATCH /v1/catalogs/" + italiaID + "/publishers/" + italiaPublisherID,
			body:                `{"description":"x"}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// GET /catalogs/:id/publishers
		{
			description:         "GET catalog publishers",
			query:               "GET /v1/catalogs/" + italiaID + "/publishers",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", data[0]["id"])
			},
		},
		{
			description:         "GET catalog publishers by alternativeId",
			query:               "GET /v1/catalogs/swiss/publishers",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))
				assert.Equal(t, "47807e0c-0613-4aea-9917-5455cc6eddad", data[0]["id"])
			},
		},
		{
			description:         "GET root catalog publishers (∅)",
			query:               "GET /v1/catalogs/%E2%88%85/publishers",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				// All publishers except the 2 assigned to named catalogs, minus the inactive one
				assert.Equal(t, 25, len(data))
			},
		},
		{
			description:         "GET publishers for nonexistent catalog",
			query:               "GET /v1/catalogs/nonexistent/publishers",
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't get Publishers","detail":"Catalog was not found","status":404}`,
		},

		// POST /catalogs/:id/software
		{
			description: "POST catalog software",
			query:       "POST /v1/catalogs/" + italiaID + "/software",
			body:        `{"url": "https://example.org/new-sw", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assertUUID(t, response["id"])
				assert.Equal(t, italiaID, response["catalogId"])
			},
		},
		{
			description: "POST catalog software by alternativeId",
			query:       "POST /v1/catalogs/italia/software",
			body:        `{"url": "https://example.org/new-sw-2", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaID, response["catalogId"])
			},
		},
		{
			description: "POST catalog software - root catalog (∅)",
			query:       "POST /v1/catalogs/%E2%88%85/software",
			body:        `{"url": "https://example.org/root-sw", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Nil(t, response["catalogId"])
			},
		},
		{
			description: "POST catalog software - catalog not found",
			query:       "POST /v1/catalogs/nonexistent/software",
			body:        `{"url": "https://example.org/x", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Software","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "POST catalog software - no token",
			query:               "POST /v1/catalogs/" + italiaID + "/software",
			body:                `{"url": "https://example.org/x", "publiccodeYml": "-"}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// PATCH /catalogs/:id/software/:softwareId
		{
			description: "PATCH catalog software",
			query:       "PATCH /v1/catalogs/" + italiaID + "/software/" + italiaSoftwareID,
			body:        `{"publiccodeYml": "updated"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaSoftwareID, response["id"])
				assert.Equal(t, "updated", response["publiccodeYml"])
			},
		},
		{
			description: "PATCH catalog software by alternativeId",
			query:       "PATCH /v1/catalogs/italia/software/" + italiaSoftwareID,
			body:        `{"publiccodeYml": "updated-via-alt"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, italiaSoftwareID, response["id"])
			},
		},
		{
			description: "PATCH catalog software - wrong catalog returns 404",
			query:       "PATCH /v1/catalogs/" + swissID + "/software/" + italiaSoftwareID,
			body:        `{"publiccodeYml": "x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"Software was not found","status":404}`,
		},
		{
			description: "PATCH catalog software - root catalog (∅)",
			query:       "PATCH /v1/catalogs/%E2%88%85/software/" + rootSoftwareID,
			body:        `{"publiccodeYml": "updated-root"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, rootSoftwareID, response["id"])
			},
		},
		{
			description: "PATCH catalog software - catalog-scoped software rejected for root catalog",
			query:       "PATCH /v1/catalogs/%E2%88%85/software/" + italiaSoftwareID,
			body:        `{"publiccodeYml": "x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"Software was not found","status":404}`,
		},
		{
			description: "PATCH catalog software - catalog not found",
			query:       "PATCH /v1/catalogs/nonexistent/software/" + italiaSoftwareID,
			body:        `{"publiccodeYml": "x"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Software","detail":"Catalog was not found","status":404}`,
		},
		{
			description:         "PATCH catalog software - no token",
			query:               "PATCH /v1/catalogs/" + italiaID + "/software/" + italiaSoftwareID,
			body:                `{"publiccodeYml": "x"}`,
			expectedCode:        401,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"token authentication failed","status":401}`,
		},

		// GET /catalogs/:id/software
		{
			description:         "GET catalog software",
			query:               "GET /v1/catalogs/" + italiaID + "/software",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))
				assert.Equal(t, "c353756e-8597-4e46-a99b-7da2e141603b", data[0]["id"])
			},
		},
		{
			description:         "GET catalog software filtered by url",
			query:               "GET /v1/catalogs/" + italiaID + "/software?url=https://1-a.example.org/code/repo",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 1, len(data))
				assert.Equal(t, italiaSoftwareID, data[0]["id"])
			},
		},
		{
			description:         "GET catalog software filtered by url excludes inactive",
			query:               "GET /v1/catalogs/%E2%88%85/software?url=https://31-a.example.org/code/repo",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description:         "GET catalog software filtered by url - not found",
			query:               "GET /v1/catalogs/" + italiaID + "/software?url=https://no.such.url.example.org",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description:         "GET root catalog software (∅)",
			query:               "GET /v1/catalogs/%E2%88%85/software",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				// All active software except the 2 assigned to named catalogs
				assert.Equal(t, 25, len(data))
			},
		},
		{
			description:         "GET software for nonexistent catalog",
			query:               "GET /v1/catalogs/nonexistent/software",
			expectedCode:        404,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't get Software","detail":"Catalog was not found","status":404}`,
		},
	}

	runTestCases(t, tests)
}

func TestCatalogSoftwareDBChecks(t *testing.T) {
	t.Run("POST catalog software persists catalogId", func(t *testing.T) {
		loadFixtures(t)

		req, err := newTestRequest(
			"POST",
			"/v1/catalogs/"+italiaID+"/software",
			strings.NewReader(`{"url":"https://example.org/cat-sw","publiccodeYml":"-"}`),
		)
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&created))
		softwareID := created["id"].(string)

		assert.Equal(t, italiaID, dbValue(t, "software", "catalog_id", "id", softwareID))
	})

	t.Run("PATCH catalog software persists publiccodeYml", func(t *testing.T) {
		loadFixtures(t)

		req, err := newTestRequest(
			"PATCH",
			fmt.Sprintf("/v1/catalogs/%s/software/%s", italiaID, italiaSoftwareID),
			strings.NewReader(`{"publiccodeYml":"patched-yml"}`),
		)
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		assert.Equal(t, "patched-yml", dbValue(t, "software", "publiccode_yml", "id", italiaSoftwareID))
	})
}

func TestCatalogDeleteDBChecks(t *testing.T) {
	t.Run("DELETE catalog removes it and its sources from DB", func(t *testing.T) {
		loadFixtures(t)

		body := `{"name": "To Delete", "sources": [{"url": "https://github.com/example/to-delete"}]}`
		req, err := newTestRequest("POST", "/v1/catalogs", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&created))
		catalogID := created["id"].(string)

		req, err = newTestRequest("DELETE", "/v1/catalogs/"+catalogID, nil)
		require.NoError(t, err)
		req.Header = map[string][]string{"Authorization": {goodToken}}

		res, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 204, res.StatusCode)

		assert.Equal(t, 0, dbCount(t, "catalogs", "id", catalogID))
		assert.Equal(t, 0, dbCount(t, "catalog_sources", "catalog_id", catalogID))
	})
}

func TestCatalogSourcesDBChecks(t *testing.T) {
	t.Run("POST stores driver when provided", func(t *testing.T) {
		loadFixtures(t)

		body := `{"name":"With Driver","sources":[{"url":"https://code.example.org/repo","driver":"custom"}]}`
		req, err := newTestRequest("POST", "/v1/catalogs", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&created))
		catalogID := created["id"].(string)

		assert.Equal(t, "custom", dbValue(t, "catalog_sources", "driver", "catalog_id", catalogID))
	})

	t.Run("POST accepts source without driver", func(t *testing.T) {
		loadFixtures(t)

		body := `{"name":"No Driver","sources":[{"url":"https://code.example.org/repo"}]}`
		req, err := newTestRequest("POST", "/v1/catalogs", strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)
	})

	t.Run("PATCH with sources replaces them", func(t *testing.T) {
		loadFixtures(t)

		const sourceURL = "https://gitlab.com/example/replaced"

		body := fmt.Sprintf(`{"sources":[{"url":"%s","driver":"gitlab"}]}`, sourceURL)
		req, err := newTestRequest("PATCH", "/v1/catalogs/"+italiaID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		assert.Equal(t, 1, dbCount(t, "catalog_sources", "catalog_id", italiaID))
		assert.Equal(t, sourceURL, dbValue(t, "catalog_sources", "url", "catalog_id", italiaID))
		assert.Equal(t, "gitlab", dbValue(t, "catalog_sources", "driver", "catalog_id", italiaID))
	})

	t.Run("PATCH persists args as JSON", func(t *testing.T) {
		loadFixtures(t)

		body := `{"sources":[{"url":"https://example.org/data.json","driver":"json","args":["$.items[*].url"]}]}`
		req, err := newTestRequest("PATCH", "/v1/catalogs/"+italiaID, strings.NewReader(body))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode)

		var catalog map[string]interface{}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&catalog))

		sources, ok := catalog["sources"].([]interface{})
		require.True(t, ok)
		require.Equal(t, 1, len(sources))

		src := sources[0].(map[string]interface{})
		assert.Equal(t, "json", src["driver"])

		args, ok := src["args"].([]interface{})
		require.True(t, ok)
		require.Equal(t, 1, len(args))
		assert.Equal(t, "$.items[*].url", args[0])
	})
}
