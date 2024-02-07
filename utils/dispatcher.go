package utils

import (
	"net/http"
)

// Dispatch500Error 500 - internal server error
func Dispatch500Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(WriteError(http.StatusInternalServerError, msg))
}

// Dispatch501Error - not implemented
func Dispatch501Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write(WriteError(http.StatusNotImplemented, msg))
}

// Dispatch405Error - method not allowed
func Dispatch405Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write(WriteError(http.StatusMethodNotAllowed, msg))
}

// Dispatch400Error - bad request
func Dispatch400Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write(WriteError(http.StatusBadRequest, msg))
}

// Dispatch401Error - unauthorized
func Dispatch401Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(WriteError(http.StatusUnauthorized, msg))
}

// Dispatch404Error - not found
func Dispatch404Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write(WriteError(http.StatusNotFound, msg))
}
