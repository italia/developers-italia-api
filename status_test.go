package main

import (
	"testing"
)

func TestStatusEndpoints(t *testing.T) {
	tests := []TestCase{
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
