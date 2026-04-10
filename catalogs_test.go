package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	italiaID = "a8e5e6d7-0b1c-4f2a-8e3d-9c4b5a6f7e8d"
	swissID  = "b9f6f7e8-1c2d-4f3b-9f4e-0d5c6b7a8f9e"
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
				assertOnlyKeys(t, first, "id", "name", "alternativeId", "active", "createdAt", "updatedAt")
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
				assertOnlyKeys(t, response, "id", "name", "alternativeId", "active", "createdAt", "updatedAt")
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
			body:        `{"name": "New Catalog"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assertUUID(t, response["id"])
				assert.Equal(t, "New Catalog", response["name"])
				assertOnlyKeys(t, response, "id", "name", "active", "createdAt", "updatedAt")
			},
		},
		{
			description: "POST catalog with alternativeId",
			query:       "POST /v1/catalogs",
			body:        `{"name": "Another Catalog", "alternativeId": "another"}`,
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
			body:        `{"name": "Dup", "alternativeId": "italia"}`,
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
			expectedBody:        `{"title":"can't create Catalog","detail":"invalid format: name is required","status":422,"validationErrors":[{"field":"name","rule":"required","value":""}]}`,
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

func TestCatalogDeleteDBChecks(t *testing.T) {
	t.Run("DELETE catalog removes it from DB when empty", func(t *testing.T) {
		loadFixtures(t)

		// Create an empty catalog to delete
		body := `{"name": "To Delete"}`
		req, err := http.NewRequest("POST", "/v1/catalogs", strings.NewReader(body))
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

		req, err = http.NewRequest("DELETE", "/v1/catalogs/"+catalogID, nil)
		require.NoError(t, err)
		req.Header = map[string][]string{"Authorization": {goodToken}}

		res, err = app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 204, res.StatusCode)

		assert.Equal(t, 0, dbCount(t, "catalogs", "id", catalogID))
	})
}
