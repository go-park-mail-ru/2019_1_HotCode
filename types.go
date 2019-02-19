package main

// Configuration для парса конфигов
type Configuration struct {
	Port int
}

// BasicUser базовые поля
type BasicUser struct {
	Username string `json:"username"`
}

//TableName имя таблицы
func (u *BasicUser) TableName() string {
	return "user"
}

// InfoUser BasicUser, расширенный служебной инфой
type InfoUser struct {
	BasicUser
	ID     int64 `json:"id" gorm:"primary_key"`
	Active bool  `json:"active" gorm:"default:true"`
}

//TableName имя таблицы
func (u *InfoUser) TableName() string {
	return "user"
}

// FormUser BasicUser, расширенный паролем, используется для входа и регистрации
type FormUser struct {
	BasicUser
	PasswordRaw     string `json:"password" gorm:"-"`
	PasswordEncoded []byte `gorm:"column:password"`
}

//TableName имя таблицы
func (u *FormUser) TableName() string {
	return "user"
}

// Validate валидация структуры,
// TODO: убрать в либу
func (u *FormUser) Validate(errors map[string]*Error) bool {
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

// User структура для gorm
type User struct {
	ID              int64  `json:"id" gorm:"primary_key"`
	Username        string `json:"username"`
	PasswordRaw     string `json:"password" gorm:"-"`
	PasswordEncoded []byte `gorm:"column:password"`
	Active          bool   `json:"active" gorm:"default:true"`
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
