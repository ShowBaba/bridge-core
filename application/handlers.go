package application

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

func (h *Handler) createApplication(c *fiber.Ctx) error {
	var input CreateApplicationPayload
	if err := c.BodyParser(&input); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(input); err != nil {
		return utils.Dispatch400Error(c, err.(validator.ValidationErrors).Error())
	}
	uid, ok := c.Locals("id").(string)
	if !ok || uid == "" {
		return utils.Dispatch400Error(c, "missing user id")
	}
	if err := h.svc.create(c.UserContext(), uid, input); err != nil {
		if errors.Is(err, ErrDuplicateName) {
			return utils.Dispatch400Error(c, "duplicate application name")
		}
		return utils.Dispatch500Error(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(utils.APIResponse{
		Status:  fiber.StatusCreated,
		Message: "application created successfully",
	})
}

func (h *Handler) updateApplication(c *fiber.Ctx) error {
	var input UpdateApplicationPayload
	if err := c.BodyParser(&input); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(input); err != nil {
		return utils.Dispatch400Error(c, err.Error())
	}
	appID := c.Params("application_id")
	if appID == "" {
		return utils.Dispatch400Error(c, "missing application_id in request")
	}
	uid, ok := c.Locals("id").(string)
	if !ok || uid == "" {
		return utils.Dispatch400Error(c, "missing user id")
	}
	if err := h.svc.update(c.UserContext(), uid, appID, input); err != nil {
		if errors.Is(err, ErrDuplicateName) {
			return utils.Dispatch400Error(c, "duplicate application name")
		}
		if errors.Is(err, ErrNotFound) {
			return utils.Dispatch404Error(c, "cannot find application")
		}
		return utils.Dispatch500Error(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "application updated successfully",
	})
}

func (h *Handler) deleteApplication(c *fiber.Ctx) error {
	appID := c.Params("application_id")
	if appID == "" {
		return utils.Dispatch400Error(c, "missing application_id in request")
	}
	uid, ok := c.Locals("id").(string)
	if !ok || uid == "" {
		return utils.Dispatch400Error(c, "missing user id")
	}
	if err := h.svc.delete(c.UserContext(), uid, appID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return utils.Dispatch404Error(c, "cannot find application")
		}
		return utils.Dispatch500Error(c, err)
	}
	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "application deleted successfully",
	})
}
