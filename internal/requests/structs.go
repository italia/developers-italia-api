package requests

type Publisher struct {
	Name string `json:"name" validate:"required,max=255"`
}
