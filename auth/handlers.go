package auth

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc}
}

func (h *Handler) login(c *fiber.Ctx) error {
	ip := c.IP()

	ctx := utils.ContextWithIP(c.UserContext(), ip)
	var input LoginPayload
	if err := c.BodyParser(&input); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}

	v := validator.New()
	if err := v.Struct(input); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return utils.Dispatch400Error(c, ve.Error())
		}
		return utils.Dispatch400Error(c, "invalid payload")
	}

	token, err := h.svc.login(ctx, input)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return utils.Dispatch400Error(c, "email is not registered")
		}
		if errors.Is(err, ErrInvalidLogin) {
			return utils.Dispatch400Error(c, "incorrect password")
		}
		return utils.Dispatch500Error(c, err)
	}

	type Token struct {
		Token string `json:"token"`
	}
	resp := utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "login successfully",
		Data:    Token{Token: token},
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}
