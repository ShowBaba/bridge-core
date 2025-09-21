package database

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

func (h *Handler) update(c *fiber.Ctx) error {
	var input UpdateDatabasePayload
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

	dbID := c.Params("database_id")
	if dbID == "" {
		return utils.Dispatch400Error(c, "invalid or missing database ID")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}

	if err := h.svc.update(c.UserContext(), uid, dbID, input); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "database or application not found")
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		case errors.Is(err, ErrDuplicateDBName):
			return utils.Dispatch400Error(c, "duplicate database name")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "database updated successfully",
	})
}

func (h *Handler) delete(c *fiber.Ctx) error {
	dbID := c.Params("database_id")
	if dbID == "" {
		return utils.Dispatch400Error(c, "invalid or missing database ID")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}

	if err := h.svc.delete(c.UserContext(), uid, dbID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "database or application not found")
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "database deleted successfully",
	})
}

func (h *Handler) add(c *fiber.Ctx) error {
	var input AddDatabasePayload
	if err := c.BodyParser(&input); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(input); err != nil {
		return utils.Dispatch400Error(c, err.(validator.ValidationErrors).Error())
	}
	appID := c.Params("application_id")
	if appID == "" {
		return utils.Dispatch400Error(c, "missing application_id in request")
	}
	uid, ok := c.Locals("id").(string)
	if !ok || uid == "" {
		return utils.Dispatch400Error(c, "missing user id")
	}
	if err := h.svc.add(c.UserContext(), uid, appID, input); err != nil {
		if errors.Is(err, ErrDuplicateDBName) {
			return utils.Dispatch400Error(c, "duplicate database name")
		}
		if errors.Is(err, ErrNotFound) {
			return utils.Dispatch404Error(c, "cannot find application")
		}
		return utils.Dispatch500Error(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(utils.APIResponse{
		Status:  fiber.StatusCreated,
		Message: "database created successfully",
	})
}
