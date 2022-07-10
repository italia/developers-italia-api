package requests

type Publisher struct {
	Name string `json:"name" validate:"required,max=255"`
}

type Log struct {
	Message string `json:"message" validate:"required"`
}
