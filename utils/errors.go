package utils

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	// ErrBadJSON некорректный JSON
	ErrBadJSON = errors.New("bad_json")
	// ErrUnauthorized не авторизован
	ErrUnauthorized = errors.New("unauthorized")
	// ErrInvalid у поля неправильный формат
	ErrInvalid = errors.New("invalid")
	// ErrRequired поле обязательно, но не было передано
	ErrRequired = errors.New("required")
	// ErrTaken это поле должно быть уникальным и уже используется
	ErrTaken = errors.New("taken")
	// ErrNotExists такой записи нет
	ErrNotExists = errors.New("not_exists")
	// ErrInternal всё очень плохо
	ErrInternal = errors.New("internal server error")
)

// ValidationError описывает ошибки валидцации
// в формате "field": "error"
type ValidationError map[string]string

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%+v", *ve)
}

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

// ErrorResponseWriter объект для записи ошибок в лог и коннект
type ErrorResponseWriter struct {
	w      http.ResponseWriter
	logger *log.Entry
}

// NewErrorResponseWriter создаёт объект для логгирования и записи ошибок
func NewErrorResponseWriter(w http.ResponseWriter, logger *log.Entry) *ErrorResponseWriter {
	return &ErrorResponseWriter{
		w:      w,
		logger: logger,
	}
}

// WriteError запись ошибки в лог и коннект
func (e *ErrorResponseWriter) WriteError(code int, err error) {
	e.logger.Error(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(code), code))
	WriteApplicationJSON(e.w, code, NewAPIError(err))
}

// WriteWarn запись ворнинга в лог и коннект
func (e *ErrorResponseWriter) WriteWarn(code int, err error) {
	e.logger.Warn(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(code), code))
	WriteApplicationJSON(e.w, code, NewAPIError(err))
}

// WriteValidationError запись ошибки валидации(по дефолту юзается http.StatusBadRequest)
func (e *ErrorResponseWriter) WriteValidationError(err *ValidationError) {
	e.logger.Warn(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(http.StatusBadRequest), http.StatusBadRequest))
	WriteApplicationJSON(e.w, http.StatusBadRequest, err)
}

func GetLogger(r *http.Request, funcName string) *log.Entry {
	token := TokenInfo(r)
	return log.WithFields(log.Fields{
		"token":  token,
		"method": funcName,
	})
}
