package controllers

import (
	"fmt"

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
	Username *string `json:"username"`
}

// InfoUser BasicUser, расширенный служебной инфой
type InfoUser struct {
	BasicUser
	ID     *int64 `json:"id"`
	Active *bool  `json:"active"`
}

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	Password *string `json:"password"`
}

// Validate валидация полей
func (fu *FormUser) Validate() *ValidationError {
	err := ValidationError{}
	if fu.Username == nil {
		err["username"] = models.ErrRequired.Error()
	}
	if fu.Password == nil {
		err["password"] = models.ErrRequired.Error()
	}

	if len(err) == 0 {
		return nil
	}

	return &err
}
