package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

// DecodeBodyJSON парсит body в переданную структуру
func DecodeBodyJSON(body io.Reader, v interface{}) error {
	decoder := json.NewDecoder(body)
	err := decoder.Decode(v)
	if err != nil {
		return err
	}

	return nil
}

// WriteApplicationJSON отправить JSON со структурой или 500, если не ок;
func WriteApplicationJSON(w http.ResponseWriter, code int, v interface{}) {
	logger := log.WithField("method", "WriteApplicationJSON")
	w.Header().Set("Content-Type", "application/json")

	respJSON, err := json.Marshal(v)
	if err != nil {
		logger.Error(err)
		code = http.StatusInternalServerError
		respJSON = []byte(fmt.Sprintf(`"message":"%s"`, err.Error()))
	}

	w.WriteHeader(code)
	_, err = w.Write(respJSON)
	if err != nil {
		logger.Error(err)
	}
}
