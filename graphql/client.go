package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

func RunGQL(c *fiber.Ctx, schema graphql.Schema, ctx context.Context) error {
	body := c.Body()
	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "failure", "message": "invalid request body"})
	}

	var (
		payload Payload
		resp    *graphql.Result
	)

	isIntrospection := false
	err := json.Unmarshal(body, &payload)
	if err == nil {
		q := payload.Query
		on := payload.OperationName
		isIntrospection = strings.Contains(q, "__schema") || strings.Contains(q, "__type") || on == "IntrospectionQuery"

		if !isIntrospection {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "failure", "message": "auth token not in header"})
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "failure", "message": "bearer token not in header"})
			}
			claim, err := utils.ValidateAuthToken(parts[1], utils.GetConfig().JWTSecretKey)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "failure", "message": fmt.Sprintf("error validating auth token token: %v", err)})
			}
			ctx = context.WithValue(ctx, utils.KeyID, claim.ID)
		}

		resp = graphql.Do(graphql.Params{
			Schema:         schema,
			RequestString:  payload.Query,
			VariableValues: payload.Variables,
			OperationName:  payload.OperationName,
			Context:        ctx,
		})
	} else {
		q := string(body)
		isIntrospection = strings.Contains(q, "__schema") || strings.Contains(q, "__type")
		resp = graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: q,
			Context:       ctx,
		})
	}

	if len(resp.Errors) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "failure", "message": fmt.Sprintf("%+v", resp.Errors)})
	}

	return c.JSON(resp)
}
