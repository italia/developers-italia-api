package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEndpoints(t *testing.T) {
	// goodToken := "Bearer v2.local.TwwHUQEi8hr2Eo881_Bs5vK9dHOR5BgEU24QRf-U7VmUwI1yOEA6mFT0EsXioMkFT_T-jjrtIJ_Nv8f6hR6ifJXUOuzWEkm9Ijq1mqSjQatD3aDqKMyjjBA"
	// badToken := "Bearer v2.local.UngfrCDNwGUw4pff2oBNoyxYvOErcbVVqLndl6nzONafUCzktaOeMSmoI7B0h62zoxXXLqTm_Phl"

	tests := []struct {
		description string

		// Test input
		route   string
		method  string
		body    string
		headers map[string][]string

		// Expected output
		expectedCode        int
		expectedBody        any
		expectedContentType string
	}{
		{
			description: "publishers get route",
			route:       "/v1/publishers",
			method:      "GET",

			expectedCode:        200,
			expectedBody:        "[]",
			expectedContentType: "application/json",
		},
		{
			description: "non existing route",
			route:       "/v1/i-dont-exist",
			method:      "GET",

			expectedCode:        404,
			expectedBody:        `{"title":"Not Found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description:         "publishers get non-existing id",
			route:               "/v1/publishers/404",
			method:              "GET",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description:         "GET status",
			route:               "/status",
			method:              "GET",
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
			// TODO: test cache headers
		},

		// Publishers
		/*{
			description: "POST publisher",
			route:       "/v1/publishers",
			method:      "POST",
			body:        `{"URL":"https://www.example.com", "email":"example@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedBody:        `{"email":"example@example.com","description":"","CodeHosting":[{"ID":1,"url":"https://www.example.com"}]}`,
			expectedContentType: "application/json",
		},
		{
			description: "POST publisher - wrong token",
			route:       "/v1/publishers",
			method:      "POST",
			body:        `{"name": "New publisher"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},*/
	}

	os.Remove("./test.db")

	os.Setenv("DATABASE_DSN", "file:./test.db")
	os.Setenv("ENVIRONMENT", "test")

	// echo -n 'test-paseto-key-dont-use-in-prod'  | base64
	os.Setenv("PASETO_KEY", "dGVzdC1wYXNldG8ta2V5LWRvbnQtdXNlLWluLXByb2Q=")

	// Setup the app as it is done in the main function
	app := Setup()

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req, _ := http.NewRequest(
				test.method,
				test.route,
				strings.NewReader(test.body),
			)
			if test.headers != nil {
				req.Header = test.headers
			}

			res, err := app.Test(req, -1)

			assert.Nil(t, err)

			assert.Equal(t, test.expectedCode, res.StatusCode)

			body, err := ioutil.ReadAll(res.Body)

			assert.Nil(t, err)

			assert.Equal(t, test.expectedBody, string(body))

			assert.Equal(t, test.expectedContentType, res.Header.Get("Content-Type"))
		})
	}
}
