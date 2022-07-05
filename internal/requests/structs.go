package requests

type Publisher struct {
	OrganizationID string `json:"organizationId" validate:"required,max=255"`
	URL            string `json:"url" validate:"required"`
	Email          string `json:"email" validate:"required,email"`
}
