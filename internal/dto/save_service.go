package dto

type SaveServiceInput struct {
	ID     string `json:"-"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Active bool   `json:"active"`
}
