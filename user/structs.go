package user

type RegisterPayload struct {
	Firstname string `json:"firstname" validate:"required"`
	Lastname  string `json:"lastname" validate:"required"`
	Email     string `json:"email" validate:"required"`
	Password  string `json:"password" validate:"required"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}

type UpdateProfilePayload struct {
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
}
