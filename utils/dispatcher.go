package utils

import (
	"net/http"
)

// 500 - internal server error
func Dispatch500Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write(WriteError(http.StatusInternalServerError, msg))
}

// 501 - not implemented
func Dispatch501Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write(WriteError(http.StatusNotImplemented, msg))
}

// 405 - method not allowed
func Dispatch405Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write(WriteError(http.StatusMethodNotAllowed, msg))
}

// 400 - bad request
func Dispatch400Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write(WriteError(http.StatusBadRequest, msg))
}

// 404 - not found
func Dispatch404Error(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write(WriteError(http.StatusNotFound, msg))
}
