package options

import "errors"

const (
	NameSiteName = "site.name"

	CodeInvalid = "options.invalid"
)

var ErrInvalidOption = errors.New("options: invalid option")

type Option struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpdateInput struct {
	Name  string
	Value string
}
