package dto

type Service struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Active bool   `json:"active"`
}

type GetServicesOutput struct {
	Services []Service
}
