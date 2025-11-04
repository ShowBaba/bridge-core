package endpoint

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/showbaba/query-bridge/bridge-core/database"
)

type ScriptInput struct {
	Lang            string `json:"lang"`
	Code            string `json:"code"`
	Enabled         *bool  `json:"enabled"`
	ScriptTimeoutMS *int   `json:"script_timeout_ms"`
	Kind            string `json:"kind"` // pre | post
}

type CreateEndpointInput struct {
	Name           string   `json:"name" validate:"required"`
	ApplicationID  string   `json:"application_id" validate:"required"`
	DatabaseID     string   `json:"database_id" validate:"required"`
	TableID        string   `json:"table_id" validate:"required"`
	Method         string   `json:"method" validate:"required"`
	Path           string   `json:"path" validate:"required"`
	Version        string   `json:"version"`
	QueryTemplate  string   `json:"query_template"`
	Columns        []string `json:"columns"`
	IsPublic       *bool    `json:"is_public"`
	LimitDefault   *uint    `json:"limit_default"`
	LimitMax       *uint    `json:"limit_max"`
	OrderBy        string   `json:"order_by"`
	OrderDirection string   `json:"order_direction"`
	TimeoutMS      *int     `json:"timeout_ms"`

	ParamSchema any `json:"param_schema"`
	QuerySchema any `json:"query_schema"`
	BodySchema  any `json:"body_schema"`

	PreScript  *ScriptInput `json:"pre_script"`
	PostScript *ScriptInput `json:"post_script"`
}

type UpdateEndpointScriptsInput struct {
	PreScripts  []ScriptInput `json:"pre_scripts"`
	PostScripts []ScriptInput `json:"post_scripts"`
}

type PreviewEndpointSQLInput struct {
	ApplicationID  string   `json:"application_id" validate:"required"`
	DatabaseID     string   `json:"database_id" validate:"required"`
	TableID        string   `json:"table_id" validate:"required"`
	Method         string   `json:"method" validate:"required"`
	Path           string   `json:"path" validate:"required"`
	Version        string   `json:"version"`
	QueryTemplate  string   `json:"query_template"`
	Columns        []string `json:"columns"`
	IsPublic       *bool    `json:"is_public"`
	LimitDefault   *uint    `json:"limit_default"`
	LimitMax       *uint    `json:"limit_max"`
	OrderBy        string   `json:"order_by"`
	OrderDirection string   `json:"order_direction"`
	TimeoutMS      *int     `json:"timeout_ms"`

	ParamSchema any `json:"param_schema"`
	QuerySchema any `json:"query_schema"`
	BodySchema  any `json:"body_schema"`
}

func (c CreateEndpointInput) GetMethod() string         { return c.Method }
func (c CreateEndpointInput) GetColumns() []string      { return c.Columns }
func (c CreateEndpointInput) GetOrderBy() string        { return c.OrderBy }
func (c CreateEndpointInput) GetOrderDirection() string { return c.OrderDirection }
func (c CreateEndpointInput) GetLimitDefault() *uint    { return c.LimitDefault }

func (p PreviewEndpointSQLInput) GetMethod() string         { return p.Method }
func (p PreviewEndpointSQLInput) GetColumns() []string      { return p.Columns }
func (p PreviewEndpointSQLInput) GetOrderBy() string        { return p.OrderBy }
func (p PreviewEndpointSQLInput) GetOrderDirection() string { return p.OrderDirection }
func (p PreviewEndpointSQLInput) GetLimitDefault() *uint    { return p.LimitDefault }

type UpdateEndpointInput struct {
	Name           string   `json:"name"`
	TableID        string   `json:"table_id"`
	Method         string   `json:"method"`
	Path           string   `json:"path"`
	QueryTemplate  string   `json:"query_template"`
	Columns        []string `json:"columns"`
	IsPublic       *bool    `json:"is_public"`
	LimitDefault   *uint    `json:"limit_default"`
	LimitMax       *uint    `json:"limit_max"`
	OrderBy        string   `json:"order_by"`
	OrderDirection string   `json:"order_direction"`
	TimeoutMS      *int     `json:"timeout_ms"`

	ParamSchema any `json:"param_schema"`
	QuerySchema any `json:"query_schema"`
	BodySchema  any `json:"body_schema"`

	PreScript  *ScriptInput `json:"pre_script"`
	PostScript *ScriptInput `json:"post_script"`
}

type ExecuteEndpointInput struct {
	Values []interface{}  `json:"values"`
	Body   map[string]any `json:"body"`
}

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "OPTIONS": true,
}
var allowedOrderDirection = map[string]bool{"DESC": true, "ASC": true}

func (input *CreateEndpointInput) ValidateMethod() error {
	if input.Method != "" && !allowedMethods[input.Method] {
		return errors.New("invalid method")
	}
	return nil
}

func (input *PreviewEndpointSQLInput) ValidateMethod() error {
	if input.Method != "" && !allowedMethods[input.Method] {
		return errors.New("invalid method")
	}
	return nil
}
func (input *UpdateEndpointInput) ValidateMethod() error {
	if input.Method != "" && !allowedMethods[input.Method] {
		return errors.New("invalid method")
	}
	return nil
}
func (input *CreateEndpointInput) ValidateOrderDirection() error {
	if input.OrderDirection != "" && !allowedOrderDirection[input.OrderDirection] {
		return errors.New("invalid order_direction")
	}
	return nil
}

func (input *PreviewEndpointSQLInput) ValidateOrderDirection() error {
	if input.OrderDirection != "" && !allowedOrderDirection[input.OrderDirection] {
		return errors.New("invalid order_direction")
	}
	return nil
}
func (input *UpdateEndpointInput) ValidateOrderDirection() error {
	if input.OrderDirection != "" && !allowedOrderDirection[input.OrderDirection] {
		return errors.New("invalid order_direction")
	}
	return nil
}

var pathRE = regexp.MustCompile(`^\/[A-Za-z0-9\-._~\/:]*$`)

func validatePath(p string) error {
	if p == "" {
		return errors.New("path is required")
	}
	if !pathRE.MatchString(p) {
		return errors.New("invalid path")
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

type PreviewScriptInput struct {
	Kind      string `json:"kind" validate:"required,oneof=pre post"`
	Lang      string `json:"lang" validate:"required,oneof=js"`
	Code      string `json:"code" validate:"required"`
	TimeoutMS *int   `json:"timeout_ms"`

	Request  map[string]any `json:"request"`  // headers, pathParams, query, body, values
	Response map[string]any `json:"response"` // status, headers, body|rows|rowCount (for GET)

}
