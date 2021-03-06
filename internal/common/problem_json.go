package common

import (
	"fmt"
)

type ProblemJSONError struct {
	Code             string            `json:"code,omitempty"`
	Title            string            `json:"title"`
	Detail           string            `json:"detail,omitempty"`
	Status           int               `json:"status"`
	ValidationErrors []ValidationError `json:"validationErrors,omitempty"`
}

func (pj ProblemJSONError) Error() string {
	return fmt.Sprintf("%s: %s. %s", pj.Code, pj.Title, pj.Detail)
}
