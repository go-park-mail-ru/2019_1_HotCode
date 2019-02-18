package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func writeApplicationJSON(w http.ResponseWriter, v interface{}) error {
	respJSON, err := json.Marshal(v)
	if err != nil {
		return err
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
