package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func writeFatalError(w http.ResponseWriter, statusCode int, logStr, respStr string) {
	log.Errorf("[Status %d] %s", statusCode, logStr)
	http.Error(w, respStr, statusCode)
}

func writeApplicationJSON(w http.ResponseWriter, v interface{}) error {
	respJSON, err := json.Marshal(v)
	if err != nil {
		writeFatalError(w, http.StatusInternalServerError,
			fmt.Sprintf("result marshal error: %s", err.Error()),
			"internal server error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respJSON)
	return nil
}

func decodeBodyJSON(body io.Reader, v interface{}) error {
	decoder := json.NewDecoder(body)
	err := decoder.Decode(v)
	if err != nil {
		return err
	}

	return nil
}
