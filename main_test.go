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
		expectedError bool
		expectedCode  int
		expectedBody  string
		method        string
	}{
		{
			description:   "publishers get route",
			route:         "/publishers",
			method:        "GET",
			expectedError: false,
			expectedCode:  200,
			expectedBody:  "[]",
		},
		{
			description:   "non existing route",
			route:         "/i-dont-exist",
			method:        "GET",
			expectedError: false,
			expectedCode:  404,
			expectedBody:  "{\"message\":\"Cannot GET /i-dont-exist\"}",
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
	}
}
