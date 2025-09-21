package endpoint

import (
	"errors"
	"fmt"

	"github.com/showbaba/query-bridge/bridge-core/database"
)

type CreateEndpointInput struct {
	Name           string   `json:"name"  validate:"required"`
	ApplicationID  string   `json:"application_id"  validate:"required"`
	TableID        string   `json:"table_id"  validate:"required"`
	Method         string   `json:"method"  validate:"required"`
	Columns        []string `json:"columns"`
	IsPublic       *bool    `json:"is_public"  validate:"required"`
	Limit          uint     `json:"limit"`
	OrderBy        string   `json:"order_by"`
	OrderDirection string   `json:"order_direction" `
}

var allowedMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"OPTIONS": true,
}

var allowedOrderDirection = map[string]bool{
	"DESC": true,
	"ASC":  true,
}

func (input *CreateEndpointInput) ValidateMethod() error {
	if !allowedMethods[input.Method] {
		return errors.New("invalid method")
	}
	return nil
}

func (input *CreateEndpointInput) ValidateOrderDirection() error {
	if !allowedOrderDirection[input.OrderDirection] {
		return errors.New("invalid method")
	}
	return nil
}

func validateOrderByColumnExist(columns []database.Column, orderBy string) error {
	for _, col := range columns {
		if col.Name == orderBy {
			return nil
		}
	}
	return fmt.Errorf("OrderBy column '%s' does not exist in the selected table", orderBy)
}

type ExecuteEndpointInput struct {
	Values []interface{} `json:"values"`
}

type UpdateEndpointInput struct {
	Name           string   `json:"name"`
	TableID        string   `json:"table_id"`
	Method         string   `json:"method"`
	Columns        []string `json:"columns"`
	IsPublic       *bool    `json:"is_public"`
	Limit          uint     `json:"limit"`
	OrderBy        string   `json:"order_by"`
	OrderDirection string   `json:"order_direction" `
}

func (input *UpdateEndpointInput) ValidateMethod() error {
	if !allowedMethods[input.Method] {
		return errors.New("invalid method")
	}
	return nil
}

func (input *UpdateEndpointInput) ValidateOrderDirection() error {
	if !allowedOrderDirection[input.OrderDirection] {
		return errors.New("invalid method")
	}
	return nil
}
