package controllers

import (
	"net/http"

	"github.com/go-park-mail-ru/2019_1_HotCode/utils"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

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

type ErrorResponseWriter struct {
	w      http.ResponseWriter
	logger *log.Entry
}

func NewErrorResponseWriter(w http.ResponseWriter, logger *log.Entry) *ErrorResponseWriter {
	return &ErrorResponseWriter{
		w:      w,
		logger: logger,
	}
}

func (e *ErrorResponseWriter) WriteError(code int, err error) {
	e.logger.Error(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(code), code))
	utils.WriteApplicationJSON(e.w, code, NewAPIError(err))
}

func (e *ErrorResponseWriter) WriteWarn(code int, err error) {
	e.logger.Warn(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(code), code))
	utils.WriteApplicationJSON(e.w, code, NewAPIError(err))
}

func (e *ErrorResponseWriter) WriteValidationError(err *ValidationError) {
	e.logger.Warn(errors.Wrapf(err, "HTTP %s[%d]", http.StatusText(http.StatusBadRequest), http.StatusBadRequest))
	utils.WriteApplicationJSON(e.w, http.StatusBadRequest, err)
}
