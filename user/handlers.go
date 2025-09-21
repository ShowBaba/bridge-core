package user

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

func (h *Handler) Register(c *fiber.Ctx) error {
	var input RegisterPayload
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

	u, err := h.svc.register(c.UserContext(), input)
	if err != nil {
		if errors.Is(err, ErrEmailInUse) {
			return utils.Dispatch400Error(c, "email already used")
		}
		return utils.Dispatch500Error(c, err)
	}

	resp := utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "user registered successfully",
		Data: map[string]string{
			"email":     u.Email,
			"firstname": u.FirstName,
			"lastname":  u.LastName,
			"id":        u.ID,
		},
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	uid, ok := c.Locals("id").(string)
	if !ok || uid == "" {
		return utils.Dispatch400Error(c, "missing user id")
	}
	u, err := h.svc.getProfile(c.UserContext(), uid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return utils.Dispatch404Error(c, "user does not exist")
		}
		return utils.Dispatch500Error(c, err)
	}
	resp := utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "fetch user profile successfully",
		Data: map[string]interface{}{
			"email":     u.Email,
			"firstname": u.FirstName,
			"lastname":  u.LastName,
			"id":        u.ID,
		},
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}
