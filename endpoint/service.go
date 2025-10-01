package endpoint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrDuplicateName  = errors.New("duplicate endpoint name")
	ErrNoColumns      = errors.New("table has no columns")
	ErrInvalidOrderBy = errors.New("invalid order by column")
	ErrInvalidAPIKey  = errors.New("invalid or missing api key")
)

type Service interface {
	create(ctx context.Context, userID, databaseID string, in CreateEndpointInput) (string, error)
	execute(c *fiber.Ctx, version, appID string, actualPath string, method string, in ExecuteEndpointInput) (interface{}, error)
	update(ctx context.Context, userID, endpointID string, in UpdateEndpointInput) (string, error)
	delete(ctx context.Context, userID, endpointID string) error
	List(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error)
	DeleteMany(ctx context.Context, ids []string) error
	previewSQL(ctx context.Context, userID, databaseID string, in PreviewEndpointSQLInput) (string, error)
}

type service struct {
	repo           Repository
	applicationSvc application.Service
	databaseSvc    database.Service
	auditSvc       audit.Service
}

func NewService(repo Repository, applicationSvc application.Service, databaseSvc database.Service, auditSvc audit.Service) Service {
	return &service{repo, applicationSvc, databaseSvc, auditSvc}
}

func (s *service) previewSQL(ctx context.Context, userID, databaseID string, in PreviewEndpointSQLInput) (string, error) {
	if err := in.ValidateMethod(); err != nil {
		return "", utils.NewBadRequest("invalid method", utils.FieldError{Field: "method", Message: err.Error()})
	}
	if err := in.ValidateOrderDirection(); err != nil {
		return "", utils.NewBadRequest("invalid order_direction", utils.FieldError{Field: "order_direction", Message: err.Error()})
	}
	if err := validatePath(in.Path); err != nil {
		return "", utils.NewBadRequest("invalid path", utils.FieldError{Field: "path", Message: err.Error()})
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: in.ApplicationID})
	if err != nil {
		return "", err
	}
	if app.UserID != userID {
		return "", ErrUnauthorized
	}

	tbl, err := s.databaseSvc.GetTable(ctx, &database.Table{ID: in.TableID, DatabaseID: databaseID})
	if err != nil {
		return "", err
	}
	cols, err := s.databaseSvc.ListColumns(ctx, database.Column{TableID: tbl.ID})
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", ErrNoColumns
	}

	if in.OrderBy != "" {
		if err := validateOrderByColumnExist(cols, in.OrderBy); err != nil {
			return "", utils.NewBadRequest("invalid order_by", utils.FieldError{Field: "order_by", Message: err.Error()})
		}
	}

	if in.QueryTemplate != "" {
		return in.QueryTemplate, nil
	}
	return buildQueryTemplate(tbl.Name, in), nil
}

func (s *service) create(ctx context.Context, userID, databaseID string, in CreateEndpointInput) (string, error) {
	in = sanitizeCreateInput(in)

	if err := in.ValidateMethod(); err != nil {
		return "", utils.NewBadRequest("invalid method", utils.FieldError{"method", err.Error()})
	}
	if err := in.ValidateOrderDirection(); err != nil {
		return "", utils.NewBadRequest("invalid order_direction", utils.FieldError{"order_direction", err.Error()})
	}
	if err := validatePath(in.Path); err != nil {
		return "", utils.NewBadRequest("invalid path", utils.FieldError{"path", err.Error()})
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: in.ApplicationID})
	if err != nil {
		return "", err
	}
	if app.UserID != userID {
		return "", ErrUnauthorized
	}

	tbl, err := s.databaseSvc.GetTable(ctx, &database.Table{ID: in.TableID, DatabaseID: databaseID})
	if err != nil {
		return "", err
	}

	cols, err := s.databaseSvc.ListColumns(ctx, database.Column{TableID: tbl.ID})
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", ErrNoColumns
	}

	if in.OrderBy != "" {
		if err := validateOrderByColumnExist(cols, in.OrderBy); err != nil {
			return "", utils.NewBadRequest("invalid order_by", utils.FieldError{Field: "order_by", Message: err.Error()})
		}
	}

	version := in.Version
	if version == "" {
		version = "v1"
	}

	if _, dup, err := s.repo.getByAppMethodPathVersion(ctx, app.ID, in.Method, in.Path, version); err != nil {
		return "", err
	} else if dup {
		return "", ErrDuplicateName
	}

	meta, err := s.loadTableMeta(ctx, in.TableID)
	if err != nil {
		return "", err
	}

	path, err := sanitizePath(in.Path)
	if err != nil {
		return "", utils.NewBadRequest("invalid path", utils.FieldError{Field: "path", Message: err.Error()})
	}
	in.Path = path

	qt := in.QueryTemplate
	if qt == "" {
		qt = buildQueryTemplate(tbl.Name, in)
	}

	defErrs := validateEndpointDefinition(defValidationOpts{
		Method:        in.Method,
		TableMeta:     meta,
		Columns:       in.Columns,
		OrderBy:       in.OrderBy,
		BodySchema:    toJSON(in.BodySchema),
		QuerySchema:   toJSON(in.QuerySchema),
		ParamSchema:   toJSON(in.ParamSchema),
		QueryTemplate: qt,
	})
	if len(defErrs) > 0 {
		fieldErrs := make([]utils.FieldError, len(defErrs))
		for i, e := range defErrs {
			fieldErrs[i] = e.ToFieldError()
		}
		return "", &utils.BadRequestError{
			Msg:     "endpoint definition invalid",
			Details: fieldErrs,
		}
	}

	url := utils.FormatPublicURL(utils.GetConfig().ServerBaseURL, version, app.Slug, in.Path)

	e := &Endpoint{
		Name:           in.Name,
		ApplicationID:  app.ID,
		DatabaseID:     in.DatabaseID,
		TableID:        in.TableID,
		Method:         in.Method,
		Path:           in.Path,
		Version:        version,
		QueryTemplate:  qt,
		Columns:        in.Columns,
		IsPublic:       utils.BoolPointer(coalesceBoolPtr(in.IsPublic, true)),
		LimitDefault:   in.LimitDefault,
		LimitMax:       in.LimitMax,
		OrderBy:        in.OrderBy,
		OrderDirection: in.OrderDirection,
		TimeoutMS:      in.TimeoutMS,
		ParamSchema:    toJSON(in.ParamSchema),
		QuerySchema:    toJSON(in.QuerySchema),
		BodySchema:     toJSON(in.BodySchema),
		UserID:         userID,
	}
	if err := s.repo.create(ctx, e); err != nil {
		return "", err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{ /* ... */ })

	return url, nil
}

func (s *service) update(ctx context.Context, userID, endpointID string, in UpdateEndpointInput) (string, error) {
	in = sanitizeUpdateInput(in)

	e, ok, err := s.repo.get(ctx, Endpoint{ID: endpointID, UserID: userID})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotFound
	}

	if in.OrderDirection != "" {
		if err := in.ValidateOrderDirection(); err != nil {
			return "", err
		}
	}
	if in.Path != "" {
		if err := validatePath(in.Path); err != nil {
			return "", err
		}
	}
	if in.Method != "" {
		if err := in.ValidateMethod(); err != nil {
			return "", err
		}
	}

	effMethod := coalesceStr(in.Method, e.Method)
	effTableID := coalesceStr(in.TableID, e.TableID)
	effColumns := func() []string {
		if len(in.Columns) > 0 {
			return in.Columns
		}
		return e.Columns
	}()
	effOrderBy := coalesceStr(in.OrderBy, e.OrderBy)
	effOrderDir := coalesceStr(in.OrderDirection, e.OrderDirection)
	var effLimit uint
	if in.LimitDefault != nil {
		effLimit = *in.LimitDefault
	} else {
		if e.LimitDefault != nil {
			effLimit = *e.LimitDefault
		} else {
			effLimit = 0
		}
	}

	tbl, err := s.databaseSvc.GetTable(ctx, &database.Table{ID: effTableID})
	if err != nil {
		return "", err
	}
	allCols, err := s.databaseSvc.ListColumns(ctx, database.Column{TableID: tbl.ID})
	if err != nil {
		return "", err
	}
	if len(allCols) == 0 {
		return "", ErrNoColumns
	}
	if len(in.Columns) > 0 {
		colSet := make(map[string]struct{}, len(allCols))
		for _, c := range allCols {
			colSet[c.Name] = struct{}{}
		}
		for _, c := range in.Columns {
			if _, ok := colSet[c]; !ok {
				return "", fmt.Errorf("column %q does not exist on table %q", c, tbl.Name)
			}
		}
	}
	if in.OrderBy != "" {
		if err := validateOrderByColumnExist(allCols, effOrderBy); err != nil {
			return "", ErrInvalidOrderBy
		}
	}

	upd := Endpoint{}
	if in.Name != "" {
		upd.Name = in.Name
	}
	if in.Path != "" {
		upd.Path = in.Path
	}
	if in.Method != "" {
		upd.Method = in.Method
	}
	if in.TableID != "" {
		upd.TableID = in.TableID
	}
	if len(in.Columns) > 0 {
		upd.Columns = in.Columns
	}
	if in.IsPublic != nil {
		upd.IsPublic = utils.BoolPointer(*in.IsPublic)
	}
	if in.LimitDefault != nil {
		upd.LimitDefault = in.LimitDefault
	}
	if in.LimitMax != nil {
		upd.LimitMax = in.LimitMax
	}
	if in.OrderBy != "" {
		upd.OrderBy = in.OrderBy
	}
	if in.OrderDirection != "" {
		upd.OrderDirection = in.OrderDirection
	}
	if in.TimeoutMS != nil {
		upd.TimeoutMS = in.TimeoutMS
	}
	if in.ParamSchema != nil {
		upd.ParamSchema = toJSON(in.ParamSchema)
	}
	if in.QuerySchema != nil {
		upd.QuerySchema = toJSON(in.QuerySchema)
	}
	if in.BodySchema != nil {
		upd.BodySchema = toJSON(in.BodySchema)
	}

	if (upd.Method != "" || upd.Path != "") &&
		(coalesceStr(upd.Method, e.Method) != e.Method || coalesceStr(upd.Path, e.Path) != e.Path) {

		effMethod := coalesceStr(upd.Method, e.Method)
		effPath := coalesceStr(upd.Path, e.Path)

		if _, dup, err := s.repo.getByAppMethodPathVersion(ctx, e.ApplicationID, effMethod, effPath, e.Version); err != nil {
			return "", err
		} else if dup {
			return "", ErrDuplicateName
		}
	}

	meta, err := s.loadTableMeta(ctx, upd.TableID)
	if err != nil {
		return "", err
	}

	path, err := sanitizePath(upd.Path)
	if err != nil {
		return "", err
	}
	upd.Path = path

	if in.QueryTemplate != "" && in.QueryTemplate != e.QueryTemplate {
		upd.QueryTemplate = in.QueryTemplate
	} else {
		upd.QueryTemplate = buildQueryTemplate(tbl.Name, CreateEndpointInput{
			Method:         effMethod,
			Columns:        effColumns,
			LimitDefault:   &effLimit,
			OrderBy:        effOrderBy,
			OrderDirection: effOrderDir,
		})
	}

	defErrs := validateEndpointDefinition(defValidationOpts{
		Method:        in.Method,
		TableMeta:     meta,
		Columns:       in.Columns,
		OrderBy:       in.OrderBy,
		BodySchema:    toJSON(in.BodySchema),
		QuerySchema:   toJSON(in.QuerySchema),
		ParamSchema:   toJSON(in.ParamSchema),
		QueryTemplate: upd.QueryTemplate,
	})
	if len(defErrs) > 0 {
		return "", fmt.Errorf("endpoint definition invalid: %+v", defErrs)
	}

	if err := s.repo.update(ctx, e, upd); err != nil {
		return "", err
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: e.ApplicationID})
	if err != nil {
		return "", err
	}

	url :=
		utils.FormatPublicURL(utils.GetConfig().ServerBaseURL,
			e.Version,
			app.Slug,
			coalesceStr(in.Path, e.Path),
		)
	return url, nil
}

func (s *service) execute(c *fiber.Ctx, version, appSlug, actualPath, method string, in ExecuteEndpointInput) (interface{}, error) {
	app, err := s.applicationSvc.Get(c.UserContext(), &application.Application{Slug: appSlug})
	if err != nil || app == nil {
		return nil, ErrNotFound
	}

	endpoints, err := s.repo.listByAppMethodAndVersion(c.UserContext(), app.ID, method, version)
	if err != nil {
		return nil, err
	}

	fmt.Println("found endpoints; ", len(endpoints))

	var matched *Endpoint
	var pathParams map[string]string
	for i := range endpoints {
		if pp, ok := matchPath(endpoints[i].Path, actualPath); ok {
			matched = &endpoints[i]
			pathParams = pp
			break
		}
	}
	fmt.Println(" found match; ", matched != nil)
	if matched == nil {
		return nil, ErrNotFound
	}

	app, err = s.applicationSvc.Get(c.UserContext(), &application.Application{ID: matched.ApplicationID})
	if err != nil {
		return nil, err
	}

	if !*matched.IsPublic {
		raw, err := utils.Decrypt(app.ApiKey, []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			return nil, err
		}
		ok, _ := validateApiKeyInRequest(c, string(raw))
		if !ok {
			return nil, ErrInvalidAPIKey
		}
	}

	tbl, err := s.databaseSvc.GetTable(c.UserContext(), &database.Table{ID: matched.TableID})
	if err != nil {
		return nil, err
	}

	schema, err := s.databaseSvc.GetSchema(c.UserContext(), database.Schema{ID: tbl.SchemaID})
	if err != nil {
		return nil, err
	}

	dbInfo, err := s.databaseSvc.Get(c.UserContext(), &database.Database{ID: schema.DatabaseID})
	if err != nil {
		return nil, err
	}

	conn, err := openSqlxConnection(dbInfo)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	queryParams := map[string][]string{}
	for k, v := range c.Queries() {
		queryParams[k] = []string{v}
	}

	sqlText, args, err := bindNamed(
		matched.QueryTemplate,
		pathParams,
		queryParams,
		in.Body,
		in.Values,
	)
	if err != nil {
		return nil, err
	}

	if matched.TimeoutMS != nil {
		if _, err := conn.Exec(fmt.Sprintf("SET LOCAL statement_timeout = '%dms'", *matched.TimeoutMS)); err != nil {
			log.Warnf("failed to set statement timeout: %v", err)
		}
	}

	if err := validateJSONParams(matched.ParamSchema, pathParams); err != nil {
		return nil, fmt.Errorf("path parameters failed validation: %w", err)
	}
	if err := validateJSONQuery(matched.QuerySchema, queryParams); err != nil {
		return nil, fmt.Errorf("query parameters failed validation: %w", err)
	}
	if strings.ToUpper(method) != "GET" {
		var bodyToValidate any
		if in.Body != nil {
			bodyToValidate = in.Body
		} else if len(in.Values) > 0 {
			bodyToValidate = in.Values
		}
		if err := validateJSONBody(matched.BodySchema, bodyToValidate); err != nil {
			return nil, fmt.Errorf("request body failed validation: %w", err)
		}
	}

	switch strings.ToUpper(matched.Method) {
	case "GET":
		res, err := executeFetchQuery(conn, sqlText, matched.Columns, args...)
		if err != nil {
			return nil, err
		}
		go s.auditSvc.Create(context.Background(), audit.LogInput{
			UserID: "external", Action: "execute", EntityType: "endpoint", EntityID: matched.ID,
			Description:   "executed endpoint (GET)",
			Metadata:      map[string]any{"rows": len(res), "path": actualPath, "method": method},
			ApplicationID: matched.ApplicationID,
		})
		return res, nil
	default:
		if err := executeInsertQuery(conn, sqlText, args); err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		go s.auditSvc.Create(context.Background(), audit.LogInput{
			UserID: "external", Action: "execute", EntityType: "endpoint", EntityID: matched.ID,
			Description:   "executed endpoint (" + method + ")",
			Metadata:      map[string]any{"path": actualPath, "method": method},
			ApplicationID: matched.ApplicationID,
		})
		return fiber.Map{"status": "ok"}, nil
	}
}

func (s *service) List(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error) {
	return s.repo.list(ctx, filter, opts)
}
func (s *service) DeleteMany(ctx context.Context, ids []string) error {
	return s.repo.deleteMany(ctx, ids)
}

func (s *service) delete(ctx context.Context, userID, endpointID string) error {
	e, ok, err := s.repo.get(ctx, Endpoint{ID: endpointID})
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}

	fmt.Println("endpoint to delete: ", e)

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: e.ApplicationID, UserID: userID})
	if err != nil {
		return err
	}
	if app.UserID != userID {
		return ErrUnauthorized
	}

	fmt.Println("found application; ", app)

	if err := s.repo.delete(ctx, &Endpoint{ID: e.ID}); err != nil {
		return err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "delete",
		EntityType:  "endpoint",
		EntityID:    e.ID,
		Description: "deleted endpoint",
		Metadata: map[string]any{
			"endpoint_id": e.ID,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: e.ApplicationID,
	})

	return nil
}

func (s *service) loadTableMeta(ctx context.Context, tableID string) (map[string]ColMeta, error) {
	tbl, err := s.databaseSvc.GetTable(ctx, &database.Table{ID: tableID})
	if err != nil {
		return nil, err
	}
	schema, err := s.databaseSvc.GetSchema(ctx, database.Schema{ID: tbl.SchemaID})
	if err != nil {
		return nil, err
	}
	dbInfo, err := s.databaseSvc.Get(ctx, &database.Database{ID: schema.DatabaseID})
	if err != nil {
		return nil, err
	}
	engine := strings.ToLower(dbInfo.DbEngine)

	cols, err := s.databaseSvc.ListColumns(ctx, database.Column{TableID: tableID})
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, ErrNoColumns
	}

	out := make(map[string]ColMeta, len(cols))
	for _, c := range cols {
		defStr := ""
		hasDefault := c.DefaultValue.Valid && c.DefaultValue.String != ""
		if hasDefault {
			defStr = strings.ToLower(c.DefaultValue.String)
		}

		auto := isAutoIncrement(engine, c, defStr)

		out[strings.ToLower(c.Name)] = ColMeta{
			Name:       c.Name,
			DataType:   c.DataType,
			Nullable:   c.IsNullable,
			HasDefault: hasDefault,
			IsPK:       c.IsPrimaryKey,
			IsAutoInc:  auto,
		}
	}
	return out, nil
}

func isAutoIncrement(engine string, c database.Column, defLower string) bool {
	dt := strings.ToLower(c.DataType)

	switch engine {
	case "postgres", "postgresql":
		if strings.Contains(defLower, "nextval(") || strings.Contains(defLower, "identity") {
			return true
		}
		if strings.Contains(dt, "serial") || strings.Contains(dt, "bigserial") {
			return true
		}
	case "mysql", "mariadb":
		if strings.Contains(defLower, "auto_increment") {
			return true
		}
		// Some setups won’t expose EXTRA here; assume non-nullable PK integer without default is auto-inc in many schemas
		if c.IsPrimaryKey && !c.IsNullable && (strings.Contains(dt, "int")) && !c.DefaultValue.Valid {
			return true
		}
	case "sqlite", "sqlite3":
		// SQLite AUTOINCREMENT is tied to INTEGER PRIMARY KEY; default usually empty
		if c.IsPrimaryKey && strings.Contains(dt, "int") {
			return true
		}
	case "sqlserver", "mssql":
		if strings.Contains(defLower, "identity") {
			return true
		}
	default:
		// Generic fallback
		if strings.Contains(defLower, "nextval(") ||
			strings.Contains(defLower, "identity") ||
			strings.Contains(defLower, "auto_increment") ||
			strings.Contains(dt, "serial") {
			return true
		}
	}
	return false
}
