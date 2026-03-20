package main

import (
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	tests := []TestCase{
		{
			query:               "GET /livez",
			expectedCode:        200,
			expectedBody:        "OK",
			expectedContentType: "text/plain; charset=utf-8",
		},
		{
			query:               "GET /readyz",
			expectedCode:        200,
			expectedBody:        "OK",
			expectedContentType: "text/plain; charset=utf-8",
		},
		{
			query:               "GET /v1/status",
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
			// TODO: test cache headers
		},
	}

	runTestCases(t, tests)
}
