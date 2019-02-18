package main

// Configuration для парса конфигов
type Configuration struct {
	Port int
}

// User структура для gorm
type User struct {
	ID              int64  `json:"id" gorm:"primary_key"`
	Username        string `json:"username"`
	PasswordRaw     string `json:"password" gorm:"-"`
	PasswordEncoded []byte `gorm:"column:password"`
	Active          bool   `json:"active" gorm:"default:true"`
}

// Validate валидация структуры,
// TODO: убрать в либу
func (u *User) Validate(errors map[string]*Error) bool {
	wasError := true
	if u.Username == "" {
		errors["username"] = &Error{
			Code:    2,
			Message: "Username is empty",
		}
		wasError = false
	}

	if u.PasswordRaw == "" {
		errors["password"] = &Error{
			Code:    2,
			Message: "Password is empty",
		}
		wasError = false
	}

	return wasError
}

//TableName имя таблицы
func (u *User) TableName() string {
	return "user"
}

// Error структура ошибки
type Error struct {
	Code        int64  `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

// FromErrors ошибки в форме:
// (поле, ошибка)
type FromErrors struct {
	Errors map[string]*Error `json:"errors"`
}
