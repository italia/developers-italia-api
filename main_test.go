package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexRoute(t *testing.T) {
	tests := []struct {
		description string

		// Test input
		route string

		// Expected output
		expectedError       bool
		expectedCode        int
		expectedBody        string
		expectedContentType string
		method              string
	}{
		{
			description:   "publishers get route",
			route:         "/publishers",
			method:        "GET",
			expectedError: false,
			expectedCode:  200,
			expectedBody:  "[]",
			expectedContentType: "application/json",
		},
		{
			description:   "non existing route",
			route:         "/i-dont-exist",
			method:        "GET",
			expectedError: false,
			expectedCode:  404,
			expectedBody:  `{"title":"Not Found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description:         "publishers get non-existing id",
			route:               "/publishers/404",
			method:              "GET",
			expectedError:       false,
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description:         "GET status",
			route:               "/status",
			method:              "GET",
			expectedError:       false,
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
			// TODO: test cache headers
		},

	}

	os.Setenv("DATABASE_DSN", "file:./test.db")
	os.Setenv("ENVIRONMENT", "test")

	// Setup the app as it is done in the main function
	app := Setup()

	for _, test := range tests {
		req, _ := http.NewRequest(
			test.method,
			test.route,
			nil,
		)

		res, err := app.Test(req, -1)

		assert.Equalf(t, test.expectedError, err != nil, test.description)

		if test.expectedError {
			continue
		}

		assert.Equalf(t, test.expectedCode, res.StatusCode, test.description)

		body, err := ioutil.ReadAll(res.Body)

		assert.Nilf(t, err, test.description)

		assert.Equalf(t, test.expectedBody, string(body), test.description)
		assert.Equalf(t, test.expectedContentType, res.Header.Get("Content-Type"), test.description)
	}
}
