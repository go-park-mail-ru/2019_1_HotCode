package controllers

import (
	"fmt"
)

// ValidationError описывает ошибки валидцации
// в формате "field": "error"
type ValidationError map[string]string

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%+v", ve)
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

// Validate валидация полей(проверка на пустсоту)
func (fu *FormUser) Validate() error {
	errs := ValidationError{}
	if fu.Username == nil {
		errs["username"] = Required
	}
	if fu.Password == nil {
		errs["password"] = Required
	}

	if len(errs) == 0 {
		return nil
	}

	return &errs
}
