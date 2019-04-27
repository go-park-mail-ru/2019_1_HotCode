package users

import (
	"github.com/HotCodeGroup/warscript-utils/utils"

	"github.com/google/uuid"
	"github.com/mailru/easyjson/opt"
)

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

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	Password string `json:"password"`
}

// Validate валидация полей
func (fu *FormUser) Validate() *utils.ValidationError {
	err := utils.ValidationError{}
	if fu.Username == "" {
		err["username"] = utils.ErrRequired.Error()
	}

	if fu.Password == "" {
		err["password"] = utils.ErrRequired.Error()
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
	err := utils.ValidationError{}
	if fu.Username.IsDefined() && fu.Username.V == "" {
		err["username"] = utils.ErrInvalid.Error()
	}

	if fu.NewPassword.IsDefined() && fu.NewPassword.V == "" {
		err["newPassword"] = utils.ErrInvalid.Error()
	}

	if fu.PhotoUUID.IsDefined() && fu.PhotoUUID.V != "" {
		if _, uuidErr := uuid.Parse(fu.PhotoUUID.V); uuidErr != nil {
			err["photo_uuid"] = utils.ErrInvalid.Error()
		}
	}

	if len(err) == 0 {
		return nil
	}

	return &err
}

const (
	// SessionInfoKey ключ, по которому в контексте
	// реквеста хранится структура юзера после валидации
	SessionInfoKey utils.ContextKey = 1
)

// SessionPayload структура, которая хранится в session storage
type SessionPayload struct {
	ID int64 `json:"id"`
}
