package api

import (
	"encoding/json"
	"net/http"
)

type JSONResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func RespondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: true,
		Data:    data,
	})
}

func RespondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(JSONResponse{
		Success: false,
		Error:   message,
	})
}
