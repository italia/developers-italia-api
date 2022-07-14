package common

type Response struct {
	Data any `json:"data,omitempty"`
}

func NewResponse(data any) *Response {
	return &Response{Data: data}
}
