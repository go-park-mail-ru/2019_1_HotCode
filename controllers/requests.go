package controllers

import (
	"fmt"

	"github.com/mailru/easyjson/opt"

	"github.com/go-park-mail-ru/2019_1_HotCode/models"
)

// ValidationError описывает ошибки валидцации
// в формате "field": "error"
type ValidationError map[string]string

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%+v", *ve)
}

// BasicUser базовые поля
type BasicUser struct {
	Username  string `json:"username"`
	PhotoUUID string `json:"photo_uuid"`
}

// InfoUser BasicUser, расширенный служебной инфой
type InfoUser struct {
	BasicUser
	ID     int64 `json:"id"`
	Active bool  `json:"active"`
}

// ScoredUser инфа о юзере расширенная его баллами
type ScoredUser struct {
	InfoUser
	Score int32 `json:"score"`
}

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	Password string `json:"password"`
}

// Validate валидация полей
func (fu *FormUser) Validate() *ValidationError {
	err := ValidationError{}
	if fu.Username == "" {
		err["username"] = models.ErrRequired.Error()
	}

	if fu.Password == "" {
		err["password"] = models.ErrRequired.Error()
	}

	if len(err) == 0 {
		return nil
	}

	return &err
}

// FormUserUpdate форма для обновления полей
type FormUserUpdate struct {
	Username    opt.String `json:"username"`
	PhotoUUID   opt.String `json:"photo_uuid"`
	OldPassword opt.String `json:"oldPassword"`
	NewPassword opt.String `json:"newPassword"`
}

// Validate валидация формы
func (fu *FormUserUpdate) Validate() error {
	err := ValidationError{}
	if fu.Username.IsDefined() && fu.Username.V == "" {
		err["username"] = models.ErrInvalid.Error()
	}

	if fu.NewPassword.IsDefined() && fu.NewPassword.V == "" {
		err["newPassword"] = models.ErrInvalid.Error()
	}

	if len(err) == 0 {
		return nil
	}

	return &err
}

// Game схема объекта игра
type Game struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}
