package controllers

const (
	// BadJSON некорректный JSON
	BadJSON = "bad_json"
	// NotExists запись не существует
	NotExists = "not_exists"
	// Required поля обязательно для заполнения
	Required = "required"
	// Invalid значение поля неверно
	Invalid = "invalid"
	// Taken уже занято
	Taken = "taken"
	// Unauthorized не авторизован
	Unauthorized = "unauthorized"
)

// APIError структура ошибки
type APIError struct {
	Message string `json:"message"`
}

// NewAPIError создаёт новый объект ошибки для создания JSON
func NewAPIError(msg string) *APIError {
	return &APIError{
		Message: msg,
	}
}
