package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	auditpkg "github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	logPkg "github.com/showbaba/query-bridge/bridge-core/log"
	log "github.com/showbaba/query-bridge/bridge-core/logger"
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
	previewScript(ctx context.Context, _ string, in PreviewScriptInput) (any, error)
	updateScripts(ctx context.Context, userID, endpointID string, in UpdateEndpointScriptsInput) ([]EndpointScript, error)
}

type service struct {
	repo           Repository
	applicationSvc application.Service
	databaseSvc    database.Service
	auditSvc       audit.Service
	logSvc         logPkg.Service
}

func NewService(repo Repository, applicationSvc application.Service,
	databaseSvc database.Service, auditSvc audit.Service, logSvc logPkg.Service) Service {
	return &service{repo, applicationSvc,
		databaseSvc, auditSvc, logSvc}
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
	created, err := s.repo.create(ctx, e)
	if err != nil {
		return "", err
	}

	if scripts := makeScriptsFromInput(in.PreScript, in.PostScript); len(scripts) > 0 {
		if err := s.repo.upsertScripts(ctx, e.ID, scripts, userID); err != nil {
			return "", err
		}
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "endpoint.create",
		EntityType:  "endpoint",
		EntityID:    created.ID,
		Description: "created endpoint",
		Metadata: map[string]any{
			"name":           created.Name,
			"endpoint_id":    created.ID,
			"application_id": created.ApplicationID,
		},
		Severity:      auditpkg.SeverityInfo,
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: created.ApplicationID,
	})

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

	if upd.Path != "" {
		path, err := sanitizePath(upd.Path)
		if err != nil {
			return "", err
		}
		upd.Path = path
	} else {
		upd.Path = e.Path
	}

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

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "endpoint.update",
		EntityType:  "endpoint",
		EntityID:    upd.ID,
		Description: "updated endpoint",
		Metadata: map[string]any{
			"name":           upd.Name,
			"endpoint_id":    upd.ID,
			"application_id": upd.ApplicationID,
		},
		Severity:      auditpkg.SeverityInfo,
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: upd.ApplicationID,
	})

	if in.PreScript != nil || in.PostScript != nil {
		if err := s.repo.upsertScripts(ctx, e.ID, makeScriptsFromInput(in.PreScript, in.PostScript), userID); err != nil {
			return "", err
		}
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

// endpoint/service.go
func (s *service) streamLog(level, source string, appID, userID string, payload any) {
	go func() {
		b, _ := json.Marshal(payload)
		_ = s.logSvc.Create(string(b), appID, userID, source, level)
	}()
}
func (s *service) execute(c *fiber.Ctx, version, appSlug, actualPath, method string, in ExecuteEndpointInput) (interface{}, error) {
	start := time.Now()
	app, err := s.applicationSvc.Get(c.UserContext(), &application.Application{Slug: appSlug})
	if err != nil || app == nil {
		return nil, ErrNotFound
	}

	endpoints, err := s.repo.listByAppMethodAndVersion(c.UserContext(), app.ID, method, version)
	if err != nil {
		return nil, err
	}

	var matched *Endpoint
	var pathParams map[string]string
	for i := range endpoints {
		if pp, ok := matchPath(endpoints[i].Path, actualPath); ok {
			matched = &endpoints[i]
			pathParams = pp
			break
		}
	}
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
	reqID := uuid.NewString()
	baseLog := func() map[string]any {
		return map[string]any{
			"reqId":      reqID,
			"appSlug":    appSlug,
			"appId":      app.ID,
			"endpointId": matched.ID,
			"version":    version,
			"method":     method,
			"path":       actualPath,
			"isPublic":   coalesceBoolPtr(matched.IsPublic, true),
			"remoteIP":   c.IP(),
			"userAgent":  c.Get("User-Agent"),
			"pathParams": pathParams,
			"query":      c.Queries(),
			"receivedAt": time.Now().UTC().Format(time.RFC3339Nano),
		}
	}

	scripts, err := s.repo.getScripts(c.UserContext(), matched.ID)
	if err != nil {
		return nil, err
	}
	preScript, hasPre := pickScript(scripts, "pre")
	postScript, hasPost := pickScript(scripts, "post")

	preReq := map[string]any{
		"headers":    c.GetReqHeaders(),
		"pathParams": pathParams,
		"query":      queryParams,
		"body":       in.Body,
		"values":     in.Values,
	}

	if hasPre {
		preStart := time.Now()
		scriptTimeout := 200
		if preScript.ScriptTimeoutMS != nil {
			scriptTimeout = *preScript.ScriptTimeoutMS
		}
		out, err := runJSScript(c.Context(), preScript.Code, preInput{
			Req: preReq,
			Env: jsEnv(version, appSlug),
		}, scriptTimeout)
		preElapsed := time.Since(preStart)

		if err != nil {
			logPayload := baseLog()
			logPayload["stage"] = "pre"
			logPayload["error"] = err.Error()
			logPayload["elapsedMs"] = preElapsed.Milliseconds()
			s.streamLog("ERROR", "PRE_SCRIPT", app.ID, app.UserID, logPayload)
			return nil, err
		}
		if out.Abort != nil {
			logPayload := baseLog()
			logPayload["stage"] = "pre"
			logPayload["abort"] = out.Abort
			logPayload["elapsedMs"] = preElapsed.Milliseconds()
			s.streamLog("WARN", "PRE_SCRIPT", app.ID, app.UserID, logPayload)
			return nil, utils.NewBadRequest(out.Abort.Message, utils.FieldError{
				Field: "pre_script", Message: "aborted",
			})
		}
		if out.Mutate != nil {
			if out.Mutate.PathParams != nil {
				pathParams = out.Mutate.PathParams
			}
			if out.Mutate.Query != nil {
				queryParams = out.Mutate.Query
			}
			if out.Mutate.Body != nil {
				if _, ok := out.Mutate.Body.(map[string]any); ok {
					in.Body = out.Mutate.Body.(map[string]any)
				}
			}
			if out.Mutate.Values != nil && len(out.Mutate.Values) > 0 {
				in.Values = out.Mutate.Values
			}
			preReq["pathParams"] = pathParams
			preReq["query"] = queryParams
			preReq["body"] = in.Body
			preReq["values"] = in.Values
		}

		logPayload := baseLog()
		logPayload["stage"] = "pre"
		logPayload["elapsedMs"] = preElapsed.Milliseconds()
		s.streamLog("INFO", "PRE_SCRIPT", app.ID, app.UserID, logPayload)
	}

	sqlText, args, err := bindNamed(
		matched.QueryTemplate,
		pathParams,
		queryParams,
		in.Body,
		in.Values,
	)
	if err != nil {
		lp := baseLog()
		lp["stage"] = "bind"
		lp["error"] = err.Error()
		s.streamLog("ERROR", "BIND", app.ID, app.UserID, lp)
		return nil, err
	}

	if matched.TimeoutMS != nil {
		log.Warn("failed to set statement timeout: %v", err)
	}

	if err := validateJSONParams(matched.ParamSchema, pathParams); err != nil {
		lp := baseLog()
		lp["stage"] = "validate"
		lp["error"] = err.Error()
		s.streamLog("ERROR", "VALIDATE", app.ID, app.UserID, lp)
		return nil, fmt.Errorf("path parameters failed validation: %w", err)
	}
	if err := validateJSONQuery(matched.QuerySchema, queryParams); err != nil {
		lp := baseLog()
		lp["stage"] = "validate"
		lp["error"] = err.Error()
		s.streamLog("ERROR", "VALIDATE", app.ID, app.UserID, lp)
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
			lp := baseLog()
			lp["stage"] = "validate"
			lp["error"] = err.Error()
			s.streamLog("ERROR", "VALIDATE", app.ID, app.UserID, lp)
			return nil, fmt.Errorf("request body failed validation: %w", err)
		}
	}

	switch strings.ToUpper(matched.Method) {
	case "GET":
		queryStart := time.Now()
		res, err := executeFetchQuery(conn, sqlText, matched.Columns, args...)
		qElapsed := time.Since(queryStart)
		totalElapsed := time.Since(start)
		if err != nil {
			lp := baseLog()
			lp["stage"] = "query"
			lp["status"] = http.StatusInternalServerError
			lp["error"] = err.Error()
			lp["sql"] = sqlText
			lp["args"] = args
			lp["queryElapsedMs"] = qElapsed.Milliseconds()
			lp["elapsedMs"] = totalElapsed.Milliseconds()
			s.streamLog("ERROR", "DB", app.ID, app.UserID, lp)
			return nil, err
		}
		queryElapsed := time.Since(queryStart)

		go func() {
			_, err := s.auditSvc.Create(context.Background(), audit.LogInput{
				UserID: "external", Action: "endpoint.execute", EntityType: "endpoint", EntityID: matched.ID,
				Description:   "executed endpoint (GET)",
				Metadata:      map[string]any{"rows": len(res), "path": actualPath, "method": method},
				ApplicationID: matched.ApplicationID,
				Severity:      auditpkg.SeverityInfo,
			})
			if err != nil {
				log.Error("error creating audit log: ", err)
			}
		}()

		if hasPost {
			to := 200
			if postScript.ScriptTimeoutMS != nil {
				to = *postScript.ScriptTimeoutMS
			}
			postStart := time.Now()
			out, err := runJSScript(c.Context(), postScript.Code, postInput{
				Res: map[string]any{"status": 200, "rows": res, "rowCount": len(res), "headers": map[string]string{}},
				Req: preReq,
				Env: jsEnv(version, appSlug),
			}, to)
			postElapsed := time.Since(postStart)

			if err != nil {
				logPayload := baseLog()
				logPayload["stage"] = "post"
				logPayload["error"] = err.Error()
				logPayload["elapsedMs"] = postElapsed.Milliseconds()
				s.streamLog("ERROR", "POST_SCRIPT", app.ID, app.UserID, logPayload)
				return nil, err
			}
			if out.Abort != nil {
				elapsed := time.Since(start)
				if err := s.recordEndpointStat(c.Context(), matched.ID, http.StatusBadRequest, int(elapsed.Milliseconds()), int(queryElapsed)); err != nil {
					log.Error("error recording endpoint stats: ", err)
				}
				logPayload := baseLog()
				logPayload["stage"] = "post"
				logPayload["abort"] = out.Abort
				logPayload["elapsedMs"] = postElapsed.Milliseconds()
				s.streamLog("WARN", "POST_SCRIPT", app.ID, app.UserID, logPayload)
				return nil, utils.NewBadRequest(out.Abort.Message, utils.FieldError{
					Field: "post_script", Message: "aborted",
				})
			}
			if out.Mutate != nil && out.Mutate.Body != nil {
				return out.Mutate.Body, nil
			}
		}
		elapsed := time.Since(start)
		if err := s.recordEndpointStat(c.Context(), matched.ID, http.StatusOK, int(elapsed.Milliseconds()), int(queryElapsed)); err != nil {
			log.Error("error recording endpoint stats: ", err)
		}
		lp := baseLog()
		lp["stage"] = "done"
		lp["status"] = http.StatusOK
		lp["rowCount"] = len(res)
		lp["sql"] = sqlText
		lp["args"] = args
		lp["queryElapsedMs"] = qElapsed.Milliseconds()
		lp["elapsedMs"] = totalElapsed.Milliseconds()
		s.streamLog("INFO", "EXEC", app.ID, app.UserID, lp)
		return res, nil
	default:
		queryStart := time.Now()
		if err := executeInsertQuery(conn, sqlText, args); err != nil {
			qElapsed := time.Since(queryStart)
			lp := baseLog()
			lp["stage"] = "query"
			lp["status"] = http.StatusInternalServerError
			lp["error"] = err.Error()
			lp["sql"] = sqlText
			lp["args"] = args
			lp["queryElapsedMs"] = qElapsed.Milliseconds()
			lp["elapsedMs"] = time.Since(start).Milliseconds()
			s.streamLog("ERROR", "DB", app.ID, app.UserID, lp)
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		queryElapsed := time.Since(queryStart)
		totalElapsed := time.Since(start)

		go func() {
			_, err := s.auditSvc.Create(context.Background(), audit.LogInput{
				UserID: "external", Action: "endpoint.execute", EntityType: "endpoint", EntityID: matched.ID,
				Description:   "executed endpoint (" + method + ")",
				Metadata:      map[string]any{"endpoint_id": matched.ID, "path": actualPath, "method": method},
				ApplicationID: matched.ApplicationID,
				Severity:      auditpkg.SeverityInfo,
			})
			if err != nil {
				log.Error("error creating audit log: ", err)
			}
		}()
		if hasPost {
			to := 200
			if postScript.ScriptTimeoutMS != nil {
				to = *postScript.ScriptTimeoutMS
			}
			postStart := time.Now()
			out, err := runJSScript(c.Context(), postScript.Code, postInput{
				Res: map[string]any{"status": 200, "body": map[string]any{"status": "ok"}, "headers": map[string]string{}},
				Req: preReq,
				Env: jsEnv(version, appSlug),
			}, to)
			postElapsed := time.Since(postStart)
			if err != nil {
				logPayload := baseLog()
				logPayload["stage"] = "post"
				logPayload["error"] = err.Error()
				logPayload["elapsedMs"] = postElapsed.Milliseconds()
				s.streamLog("ERROR", "POST_SCRIPT", app.ID, app.UserID, logPayload)
				return nil, err
			}
			if out.Abort != nil {
				elapsed := time.Since(start)

				if err := s.recordEndpointStat(c.Context(), matched.ID, http.StatusBadRequest, int(elapsed.Milliseconds()), int(queryElapsed)); err != nil {
					log.Error("error recording endpoint stats: ", err)
				}
				logPayload := baseLog()
				logPayload["stage"] = "post"
				logPayload["abort"] = out.Abort
				logPayload["elapsedMs"] = postElapsed.Milliseconds()
				s.streamLog("WARN", "POST_SCRIPT", app.ID, app.UserID, logPayload)
				return nil, utils.NewBadRequest(out.Abort.Message, utils.FieldError{
					Field: "post_script", Message: "aborted",
				})
			}
			if out.Mutate != nil && out.Mutate.Body != nil {
				return out.Mutate.Body, nil
			}
		}
		elapsed := time.Since(start)

		if err := s.recordEndpointStat(c.Context(), matched.ID, http.StatusOK, int(elapsed.Milliseconds()), int(queryElapsed)); err != nil {
			log.Error("error recording endpoint stats: ", err)
		}
		lp := baseLog()
		lp["stage"] = "done"
		lp["status"] = http.StatusOK
		lp["sql"] = sqlText
		lp["args"] = args
		lp["queryElapsedMs"] = queryElapsed.Milliseconds()
		lp["elapsedMs"] = totalElapsed.Milliseconds()
		s.streamLog("INFO", "EXEC", app.ID, app.UserID, lp)
		return fiber.Map{"status": "ok"}, nil
	}
}

func (s *service) previewScript(ctx context.Context, _ string, in PreviewScriptInput) (any, error) {
	kind := strings.ToLower(strings.TrimSpace(in.Kind))
	lang := strings.ToLower(strings.TrimSpace(in.Lang))
	code := strings.TrimSpace(in.Code)

	if kind != "pre" && kind != "post" {
		return nil, utils.NewBadRequest("invalid kind", utils.FieldError{Field: "kind", Message: "must be 'pre' or 'post'"})
	}
	if lang != "js" {
		return nil, utils.NewBadRequest("invalid lang", utils.FieldError{Field: "lang", Message: "only 'js' is supported"})
	}
	if code == "" {
		return nil, utils.NewBadRequest("invalid code", utils.FieldError{Field: "code", Message: "code is required"})
	}

	to := 200
	if in.TimeoutMS != nil && *in.TimeoutMS > 0 {
		to = *in.TimeoutMS
	}

	env := map[string]string{
		"version":  "v1",
		"app_slug": "playground",
	}

	switch kind {
	case "pre":
		req := in.Request
		if req == nil {
			req = map[string]any{}
		}

		out, err := runJSScript(ctx, code, preInput{
			Req: req,
			Env: env,
		}, to)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"abort":  out.Abort,
			"mutate": out.Mutate,
		}, nil

	case "post":
		req := in.Request
		if req == nil {
			req = map[string]any{}
		}
		res := in.Response
		if res == nil {
			res = map[string]any{"status": 200, "headers": map[string]string{}}
		}

		out, err := runJSScript(ctx, code, postInput{
			Req: req,
			Res: res,
			Env: env,
		}, to)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"abort":  out.Abort,
			"mutate": out.Mutate,
		}, nil
	}

	return nil, utils.NewBadRequest("invalid kind", utils.FieldError{Field: "kind", Message: "unsupported"})
}

func (s *service) recordEndpointStat(ctx context.Context, endpointId string, responseStatusCode, executionTime, queryElapsed int) error {
	return s.repo.createEndpointStat(ctx, &EndpointStats{
		EndpointID:           endpointId,
		ResponseStatusCode:   responseStatusCode,
		ExecutionTimeMS:      executionTime,
		QueryExecutionTimeMS: queryElapsed,
	})

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
		Action:      "endpoint.delete",
		EntityType:  "endpoint",
		EntityID:    e.ID,
		Description: "deleted endpoint",
		Metadata: map[string]any{
			"endpoint_id": e.ID,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: e.ApplicationID,
		Severity:      auditpkg.SeverityWarn,
	})

	return nil
}

func (s *service) updateScripts(ctx context.Context, userID, endpointID string, in UpdateEndpointScriptsInput) ([]EndpointScript, error) {
	e, ok, err := s.repo.get(ctx, Endpoint{ID: endpointID})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	app, err := s.applicationSvc.Get(ctx, &application.Application{ID: e.ApplicationID})
	if err != nil {
		return nil, err
	}
	if app.UserID != userID {
		return nil, ErrUnauthorized
	}

	makeBool := func(b bool) *bool { return &b }
	validateKind := func(k string) bool { return k == "pre" || k == "post" }
	validateLang := func(l string) bool { return strings.ToLower(l) == "js" }

	var toCreate []EndpointScript
	collect := func(kind string, src []ScriptInput) error {
		for i, sIn := range src {
			k := kind
			if sIn.Kind != "" {
				k = sIn.Kind
			}
			if !validateKind(k) {
				return utils.NewBadRequest("invalid script kind", utils.FieldError{
					Field:   fmt.Sprintf("%s_scripts[%d].kind", kind, i),
					Message: "must be 'pre' or 'post'",
				})
			}
			if !validateLang(sIn.Lang) {
				return utils.NewBadRequest("invalid script lang", utils.FieldError{
					Field:   fmt.Sprintf("%s_scripts[%d].lang", kind, i),
					Message: "only 'js' is supported",
				})
			}
			if len(sIn.Code) == 0 {
				return utils.NewBadRequest("script code required", utils.FieldError{
					Field:   fmt.Sprintf("%s_scripts[%d].code", kind, i),
					Message: "cannot be empty",
				})
			}
			enabled := makeBool(true)
			if sIn.Enabled != nil {
				enabled = sIn.Enabled
			}
			toCreate = append(toCreate, EndpointScript{
				EndpointID:      e.ID,
				Kind:            k,
				Lang:            strings.ToLower(sIn.Lang),
				Code:            sIn.Code,
				Enabled:         *enabled,
				ScriptTimeoutMS: sIn.ScriptTimeoutMS,
			})
		}
		return nil
	}

	if err := collect("pre", in.PreScripts); err != nil {
		return nil, err
	}
	if err := collect("post", in.PostScripts); err != nil {
		return nil, err
	}

	tx := s.repo.(*repository).db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Where("endpoint_id = ? AND deleted_at IS NULL", e.ID).Delete(&EndpointScript{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if len(toCreate) > 0 {
		if err := tx.Create(&toCreate).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:        userID,
		Action:        "endpoint.scripts.update",
		EntityType:    "endpoint",
		EntityID:      e.ID,
		Description:   "updated endpoint scripts",
		Metadata:      map[string]any{"pre_count": len(in.PreScripts), "post_count": len(in.PostScripts)},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: e.ApplicationID,
		Severity:      auditpkg.SeverityInfo,
	})

	return s.repo.listScriptsByEndpoint(ctx, e.ID)
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

func makeScriptsFromInput(pre *ScriptInput, post *ScriptInput) []EndpointScript {
	var out []EndpointScript
	if pre != nil && strings.TrimSpace(pre.Code) != "" {
		en := false
		if pre.Enabled != nil {
			en = *pre.Enabled
		}
		out = append(out, EndpointScript{
			Kind:            "pre",
			Lang:            coalesceStr(strings.TrimSpace(pre.Lang), "js"),
			Code:            pre.Code,
			Enabled:         en,
			ScriptTimeoutMS: pre.ScriptTimeoutMS,
		})
	}
	if post != nil && strings.TrimSpace(post.Code) != "" {
		en := false
		if post.Enabled != nil {
			en = *post.Enabled
		}
		out = append(out, EndpointScript{
			Kind:            "post",
			Lang:            coalesceStr(strings.TrimSpace(post.Lang), "js"),
			Code:            post.Code,
			Enabled:         en,
			ScriptTimeoutMS: post.ScriptTimeoutMS,
		})
	}
	return out
}

func pickScript(scripts []EndpointScript, kind string) (EndpointScript, bool) {
	for _, sc := range scripts {
		if sc.Kind == kind && sc.Enabled && strings.ToLower(sc.Lang) == "js" && strings.TrimSpace(sc.Code) != "" {
			return sc, true
		}
	}
	return EndpointScript{}, false
}
