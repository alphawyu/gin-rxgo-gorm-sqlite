package model

type UserWrapper struct {
	User User `json:"user"`
}

type User struct {
	Username string  `form:"username" json:"username" binding:"required,alphanum,min=4,max=255"`
	Email    string  `form:"email" json:"email" binding:"required,email"`
	Password string  `form:"password" json:"password" binding:"required,min=8,max=255"`
	Bio      string  `form:"bio" json:"bio" binding:"max=1024"`
	Image    *string `form:"image" json:"image" binding:"omitempty,url"`
}

type LoginUserWrapper struct {
	LoginUser LoginUser `json:"user"`
}

type LoginUser struct {
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=8,max=255"`
}

type UpdateUserWrapper struct {
	User UpdateUser `json:"user"`
}

type UpdateUser struct {
	Username string  `form:"username" json:"username" binding:"omitempty,alphanum,min=4,max=255"`
	Email    string  `form:"email" json:"email"`
	Password string  `form:"password" json:"password" binding:"omitempty,min=8,max=255"`
	Bio      string  `form:"bio" json:"bio" binding:"omitempty,max=1024"`
	Image    *string `form:"image" json:"image" binding:"omitempty,url"`
}

type UserResponse struct {
	Username string  `json:"username,omitempty"`
	Email    string  `json:"email,omitempty"`
	Bio      string  `json:"bio,omitempty"`
	Image    *string `json:"image,omitempty"`
	Token    string  `json:"token,omitempty"`
}

type ProfileResponse struct {
	ID        uint    `json:"-"`
	Username  string  `json:"username,omitempty"`
	Bio       string  `json:"bio,omitempty"`
	Image     *string `json:"image,omitempty"`
	Following bool    `json:"following"`
}
