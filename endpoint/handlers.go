package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc} }

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
	if in.DatabaseID == "" {
		return utils.Dispatch400Error(c, "missing database_id")
	}

	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}

	url, err := h.svc.create(c.UserContext(), uid, in.DatabaseID, in)
	if err != nil {
		var bre *utils.BadRequestError
		switch {
		case errors.As(err, &bre):
			return c.Status(fiber.StatusBadRequest).JSON(utils.APIResponse{
				Status:  fiber.StatusBadRequest,
				Message: bre.Msg,
				Data:    map[string]any{"errors": bre.Details},
			})
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "cannot find resource")
		case errors.Is(err, ErrDuplicateName):
			return utils.Dispatch400Error(c, "endpoint with same method+path+version exists")
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
		return utils.Dispatch400Error(c, "missing endpoint_id")
	}

	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}

	url, err := h.svc.update(c.UserContext(), uid, id, in)
	if err != nil {
		var bre *utils.BadRequestError
		switch {
		case errors.As(err, &bre):
			return c.Status(fiber.StatusBadRequest).JSON(utils.APIResponse{
				Status:  fiber.StatusBadRequest,
				Message: bre.Msg,
				Data:    map[string]any{"errors": bre.Details},
			})
		case errors.Is(err, ErrDuplicateName):
			return utils.Dispatch400Error(c, "endpoint with same method+path+version exists")
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
func (h *Handler) execute(c *fiber.Ctx) error {
	appSlug := c.Params("app_slug")
	actual := "/" + c.Params("*")
	method := c.Method()
	version := c.Params("version")
	var in ExecuteEndpointInput
	_ = c.BodyParser(&in)

	if len(in.Body) == 0 && len(in.Values) == 0 && len(c.Body()) > 0 {
		b := bytes.TrimSpace(c.Body())
		if len(b) > 0 && b[0] == '{' {
			var m map[string]any
			if json.Unmarshal(b, &m) == nil {
				in.Body = m
			}
		} else if len(b) > 0 && b[0] == '[' {
			var arr []any
			if json.Unmarshal(b, &arr) == nil {
				in.Values = arr
			}
		}
	}

	data, err := h.svc.execute(c, version, appSlug, actual, method, in)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidAPIKey):
			return utils.Dispatch401Error(c, "invalid/missing api_key")
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

func (h *Handler) previewSQL(c *fiber.Ctx) error {
	var in PreviewEndpointSQLInput
	if err := c.BodyParser(&in); err != nil {
		return utils.Dispatch400Error(c, "invalid body")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}
	dbID := in.DatabaseID
	if dbID == "" {
		return utils.Dispatch400Error(c, "missing database_id")
	}

	sqlText, err := h.svc.previewSQL(c.UserContext(), uid, dbID, in)
	if err != nil {
		var bre *utils.BadRequestError
		switch {
		case errors.As(err, &bre):
			return c.Status(fiber.StatusBadRequest).JSON(utils.APIResponse{
				Status:  fiber.StatusBadRequest,
				Message: bre.Msg,
				Data:    map[string]any{"errors": bre.Details},
			})
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		case errors.Is(err, ErrNoColumns):
			return utils.Dispatch400Error(c, "table has no columns")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "ok",
		Data:    map[string]any{"sql": sqlText},
	})
}

func (h *Handler) delete(c *fiber.Ctx) error {
	endpointID := c.Params("endpoint_id")
	if endpointID == "" {
		return utils.Dispatch400Error(c, "invalid or missing endpoint ID")
	}
	uid, _ := c.Locals("id").(string)
	if uid == "" {
		return utils.Dispatch401Error(c, "unauthorized")
	}

	if err := h.svc.delete(c.UserContext(), uid, endpointID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return utils.Dispatch404Error(c, "endpoint or application not found")
		case errors.Is(err, ErrUnauthorized):
			return utils.Dispatch401Error(c, "unauthorized")
		default:
			return utils.Dispatch500Error(c, err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(utils.APIResponse{
		Status:  fiber.StatusOK,
		Message: "endpoint deleted successfully",
	})
}
