package present

import "github.com/vito/bass-loop/pkg/models"

type User struct {
	Login string `json:"login"`
	URL   string `json:"url"`
}

func NewUser(user *models.User) *User {
	return &User{
		Login: user.Login,
		URL:   "https://github.com/" + user.Login,
	}
}
