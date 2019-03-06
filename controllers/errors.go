package controllers

import "errors"

var (
	// ErrBadJSON некорректный JSON
	ErrBadJSON = errors.New("bad_json")
	// ErrUnauthorized не авторизован
	ErrUnauthorized = errors.New("unauthorized")
)

// APIError структура ошибки
type APIError struct {
	Message string `json:"message"`
}

// NewAPIError создаёт новый объект ошибки для создания JSON
func NewAPIError(err error) *APIError {
	return &APIError{
		Message: err.Error(),
	}
}
