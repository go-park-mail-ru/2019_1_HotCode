package utils

import (
	"encoding/json"
	"io"
	"net/http"
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
	w.Header().Set("Content-Type", "application/json")

	respJSON, err := json.Marshal(v)
	if err != nil {
		code = http.StatusInternalServerError
		respJSON, _ = json.Marshal(&struct {
			Message string `json:"message"`
		}{
			Message: err.Error(),
		})
	}

	w.WriteHeader(code)
	w.Write(respJSON)
}
