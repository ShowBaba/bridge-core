package utils

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

func Dispatch500Error(c *fiber.Ctx, err error) error {
	// apitoolkit.ReportError(c.UserContext(), err)
	return c.Status(fiber.StatusInternalServerError).JSON(APIResponse{
		Status:  fiber.StatusInternalServerError,
		Message: err.Error(),
	})
}

func Dispatch501Error(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusNotImplemented).JSON(APIResponse{
		Status:  fiber.StatusNotImplemented,
		Message: msg,
	})
}

func Dispatch405Error(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusMethodNotAllowed).JSON(APIResponse{
		Status:  fiber.StatusMethodNotAllowed,
		Message: msg,
	})
}

func Dispatch400Error(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusBadRequest).JSON(APIResponse{
		Status:  fiber.StatusBadRequest,
		Message: msg,
	})
}

func Dispatch401Error(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(APIResponse{
		Status:  fiber.StatusUnauthorized,
		Message: msg,
	})
}

func Dispatch404Error(c *fiber.Ctx, msg string) error {
	return c.Status(fiber.StatusNotFound).JSON(APIResponse{
		Status:  fiber.StatusNotFound,
		Message: msg,
	})
}

var ErrBadRequest = errors.New("bad request")

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type BadRequestError struct {
	Msg     string       `json:"message"`
	Details []FieldError `json:"details,omitempty"`
}

func (e *BadRequestError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return ErrBadRequest.Error()
}
func (e *BadRequestError) Unwrap() error { return ErrBadRequest }

func NewBadRequest(msg string, details ...FieldError) *BadRequestError {
	return &BadRequestError{Msg: msg, Details: details}
}
