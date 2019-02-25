package models

//OK everything is fine
const OK = 0

//DB errors pack 7**
const (
	InternalDatabase = 700 + iota
	InternalStorage
	RowNotFound
	FailedToValidate
	AlreadyUsed
	CantCreate
	CantSave
	PasswordCrypt
)

// BasePack 8**
const (
	WrongJSON = 800 + iota
)

// Auth pack 9**
const (
	WrongPassword = 900 + iota
	NotActive
)

// Error структура ошибки
type Error struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
}

// Errors struct for errors:
// (поле, ошибка)
// others - ошибки, которые нельзя отнести к полям
type Errors struct {
	Other  *Error            `json:"other,omitempty"`
	Fields map[string]*Error `json:"fields,omitempty"`
}
