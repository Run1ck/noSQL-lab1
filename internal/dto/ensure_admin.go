package dto

type EnsureAdminInput struct {
	Login    string
	Password string
}

type EnsureAdminOutput struct {
	Created bool
}
