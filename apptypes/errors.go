package apptypes

//OK everything is fine
const OK = 0

// Auth pack 1**
const (
	WrongPassword = 100 + iota
	NotActive
)

// BasePack 2**
const (
	WrongJSON = 200 + iota
)

//DB errors pack 5**
const (
	InternalDatabase = 500 + iota
	InternalStorage
	RowNotFound
	FailedToValidate
	AlreadyUsed
	CantCreate
	CantSave
)

// Error структура ошибки
type Error struct {
	Code        int64  `json:"code"`
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
