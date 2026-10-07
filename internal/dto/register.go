package dto

type RegisterInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
