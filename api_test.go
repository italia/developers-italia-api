package main

import (
	"testing"
)

func TestApi(t *testing.T) {
	tests := []TestCase{
		{
			description: "non existing route",
			query:       "GET /v1/i-dont-exist",

			expectedCode:        404,
			expectedBody:        `{"title":"Not Found","detail":"Cannot GET /v1/i-dont-exist","status":404}`,
			expectedContentType: "application/problem+json",
		},
	}

	runTestCases(t, tests)
}
