package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

type CodeError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *CodeError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}
func (e *CodeError) Unwrap() error { return e.Err }
func BadRequest(code, msg string) error {
	return &CodeError{Code: code, Message: msg, Status: http.StatusBadRequest}
}
func NotFound(code, msg string) error {
	return &CodeError{Code: code, Message: msg, Status: http.StatusNotFound}
}
func Conflict(code, msg string) error {
	return &CodeError{Code: code, Message: msg, Status: http.StatusConflict}
}

type errorBody struct {
	Error struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"error"`
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "internal server error"
	var ce *CodeError
	if errors.As(err, &ce) {
		status, code, message = ce.Status, ce.Code, ce.Message
	}
	b := errorBody{}
	b.Error.Code, b.Error.Message = code, message
	b.Error.RequestID = RequestID(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(b)
}
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
