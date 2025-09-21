package endpoint

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

func (h *Handler) create(c *fiber.Ctx) error {
	var in CreateEndpointInput
	if err := c.BodyParser(&in); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(in); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return utils.Dispatch400Error(c, ve.Error())
		}
		return utils.Dispatch400Error(c, "invalid payload")
	}
	dbID := c.Params("database_id")
	if dbID == "" {
		return utils.Dispatch400Error(c, "missing database_id in request")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}
	url, err := h.svc.create(c.UserContext(), uid, dbID, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "cannot find resource")
		case errors.Is(err, ErrDuplicateName):
			return utils.Dispatch400Error(c, "endpoint with name already exist in this application and table")
		case errors.Is(err, ErrNoColumns):
			return utils.Dispatch400Error(c, "table has no columns")
		case errors.Is(err, ErrInvalidOrderBy):
			return utils.Dispatch400Error(c, "invalid order_by column")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}
	return c.Status(fiber.StatusCreated).JSON(utils.APIResponse{
		Status:  fiber.StatusCreated,
		Message: "endpoint created successfully",
		Data:    map[string]string{"url": url},
	})
}

func (h *Handler) execute(c *fiber.Ctx) error {
	var in ExecuteEndpointInput
	if err := c.BodyParser(&in); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(in); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return utils.Dispatch400Error(c, ve.Error())
		}
		return utils.Dispatch400Error(c, "invalid payload")
	}
	id := c.Params("identifier")
	if id == "" {
		return utils.Dispatch400Error(c, "missing identifier in request")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}
	data, err := h.svc.execute(c, uid, id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAPIKey):
			return utils.Dispatch401Error(c, "invalid/missing api_key in request header")
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "cannot find resource")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}
	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "query executed successfully",
		Data:    data,
	})
}

func (h *Handler) update(c *fiber.Ctx) error {
	var in UpdateEndpointInput
	if err := c.BodyParser(&in); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	v := validator.New()
	if err := v.Struct(in); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return utils.Dispatch400Error(c, ve.Error())
		}
		return utils.Dispatch400Error(c, "invalid payload")
	}
	id := c.Params("endpoint_id")
	if id == "" {
		return utils.Dispatch400Error(c, "missing endpoint_id in request")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}
	url, err := h.svc.update(c.UserContext(), uid, id, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateName):
			return utils.Dispatch400Error(c, "endpoint with name already exist in this application and table")
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "cannot find resource")
		case errors.Is(err, ErrNoColumns):
			return utils.Dispatch400Error(c, "table has no columns")
		case errors.Is(err, ErrInvalidOrderBy):
			return utils.Dispatch400Error(c, "invalid order_by column")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}
	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "endpoint updated successfully",
		Data:    map[string]string{"url": url},
	})
}
