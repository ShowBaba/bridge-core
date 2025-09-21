package endpoint

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	execute(ctx *fiber.Ctx, userID, identifier string, in ExecuteEndpointInput) (interface{}, error)
	update(ctx context.Context, userID, endpointID string, in UpdateEndpointInput) (string, error)
	List(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error)
	DeleteMany(ctx context.Context, ids []string) error
}

type service struct {
	repo           Repository
	applicationSvc application.Service
	databaseSvc    database.Service
	auditSvc       audit.Service
}

func NewService(repo Repository, applicationSvc application.Service, databaseSvc database.Service, auditSvc audit.Service) Service {
	return &service{repo: repo, applicationSvc: applicationSvc, databaseSvc: databaseSvc, auditSvc: auditSvc}
}

func (s *service) List(ctx context.Context, filter Endpoint, opts utils.ListOpts) ([]Endpoint, error) {
	return s.repo.list(ctx, filter, opts)
}

func (s *service) DeleteMany(ctx context.Context, ids []string) error {
	if err := s.repo.deleteMany(ctx, ids); err != nil {
		return err
	}
	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      "",
		Action:      "delete_many",
		EntityType:  "endpoint",
		EntityID:    "",
		Description: fmt.Sprintf("deleted %d endpoints", len(ids)),
		Metadata: map[string]any{
			"ids": ids,
		},
		IPAddress: utils.GetIPAddressFromCtx(ctx),
	})
	return nil
}

func (s *service) create(ctx context.Context, userID, databaseID string, in CreateEndpointInput) (string, error) {
	if err := in.ValidateMethod(); err != nil {
		return "", err
	}
	if in.OrderDirection != "" {
		if err := in.ValidateOrderDirection(); err != nil {
			return "", err
		}
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
			return "", ErrInvalidOrderBy
		}
	}

	if _, dup, err := s.repo.get(ctx, Endpoint{
		ApplicationID: app.ID,
		TableID:       tbl.ID,
		Name:          in.Name,
	}); err != nil {
		return "", err
	} else if dup {
		return "", ErrDuplicateName
	}

	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	url := fmt.Sprintf("%s/endpoint/execute/%s", utils.GetConfig().ServerBaseURL, id)

	var q string
	switch in.Method {
	case "GET":
		if len(in.Columns) == 0 {
			q = fmt.Sprintf("SELECT * FROM %s", tbl.Name)
		} else {
			q = fmt.Sprintf("SELECT %s FROM %s", strings.Join(in.Columns, ", "), tbl.Name)
		}
		if in.OrderBy != "" {
			q += fmt.Sprintf(" ORDER BY %s", in.OrderBy)
		}
		if in.OrderDirection != "" {
			q += fmt.Sprintf(" %s", in.OrderDirection)
		}
		if in.Limit != 0 {
			q += fmt.Sprintf(" LIMIT %d", in.Limit)
		}
	case "POST":
		vals := make([]string, len(in.Columns))
		for i := range in.Columns {
			vals[i] = fmt.Sprintf("$%d", i+1)
		}
		q = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tbl.Name, strings.Join(in.Columns, ", "), strings.Join(vals, ", "))
	}

	e := &Endpoint{
		Name:           in.Name,
		ApplicationID:  in.ApplicationID,
		TableID:        in.TableID,
		Query:          q,
		UserID:         userID,
		IdentifierUUID: id,
		Method:         in.Method,
		Limit:          in.Limit,
		IsPublic:       utils.BoolPointer(coalesceBoolPtr(in.IsPublic, true)),
		OrderBy:        in.OrderBy,
		OrderDirection: in.OrderDirection,
		Columns:        in.Columns,
		DatabaseID:     databaseID,
	}
	if err := s.repo.create(ctx, e); err != nil {
		return "", err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "create",
		EntityType:  "endpoint",
		EntityID:    e.ID,
		Description: "created endpoint",
		Metadata: map[string]any{
			"name":            e.Name,
			"method":          e.Method,
			"table_id":        e.TableID,
			"database_id":     e.DatabaseID,
			"application_id":  e.ApplicationID,
			"columns":         e.Columns,
			"limit":           e.Limit,
			"order_by":        e.OrderBy,
			"order_direction": e.OrderDirection,
			"is_public":       e.IsPublic,
			"url":             url,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: e.ApplicationID,
	})

	return url, nil
}

func (s *service) execute(ctx *fiber.Ctx, userID, identifier string, in ExecuteEndpointInput) (interface{}, error) {
	e, ok, err := s.repo.get(ctx.UserContext(), Endpoint{IdentifierUUID: identifier, UserID: userID})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}

	app, err := s.applicationSvc.Get(ctx.UserContext(), &application.Application{ID: e.ApplicationID, UserID: userID})
	if err != nil {
		return nil, err
	}

	if !*e.IsPublic {
		raw, err := utils.Decrypt(app.ApiKey, []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			return nil, err
		}
		if ok, err := validateApiKeyInRequest(ctx, string(raw)); err != nil || !ok {
			return nil, ErrInvalidAPIKey
		}
	}

	tbl, err := s.databaseSvc.GetTable(ctx.UserContext(), &database.Table{ID: e.TableID})
	if err != nil {
		return nil, err
	}

	schema, err := s.databaseSvc.GetSchema(ctx.UserContext(), database.Schema{ID: tbl.SchemaID})
	if err != nil {
		return nil, err
	}

	dbInfo, err := s.databaseSvc.Get(ctx.UserContext(), &database.Database{ID: schema.DatabaseID})
	if err != nil {
		return nil, err
	}

	conn, err := openSqlxConnection(dbInfo)
	if err != nil {
		return nil, err
	}
	defer func(conn *sqlx.DB) {
		err := conn.Close()
		if err != nil {
			log.Errorf("failed to close connection to database: %v", err)
		}
	}(conn)

	switch e.Method {
	case "POST":
		if len(in.Values) != len(e.Columns) {
			return nil, errors.New("invalid number of values in request")
		}
		if err := executeInsertQuery(conn, e.Query, in.Values); err != nil {
			return nil, fmt.Errorf("failed to execute query: %w", err)
		}
		_, _ = s.auditSvc.Create(ctx.UserContext(), audit.LogInput{
			UserID:      userID,
			Action:      "execute",
			EntityType:  "endpoint",
			EntityID:    e.ID,
			Description: "executed endpoint (POST)",
			Metadata: map[string]any{
				"values_len": len(in.Values),
			},
			IPAddress:     ctx.IP(),
			ApplicationID: e.ApplicationID,
		})
		return nil, nil
	case "GET":
		result, err := executeFetchQuery(conn, e.Query, e.Columns)
		if err != nil {
			return nil, err
		}
		_, _ = s.auditSvc.Create(ctx.UserContext(), audit.LogInput{
			UserID:      userID,
			Action:      "execute",
			EntityType:  "endpoint",
			EntityID:    e.ID,
			Description: "executed endpoint (GET)",
			Metadata: map[string]any{
				"rows": len(result),
			},
			IPAddress:     ctx.IP(),
			ApplicationID: e.ApplicationID,
		})
		return result, nil
	}
	return nil, nil
}

func (s *service) update(ctx context.Context, userID, endpointID string, in UpdateEndpointInput) (string, error) {
	if in.OrderDirection != "" {
		if err := in.ValidateOrderDirection(); err != nil {
			return "", err
		}
	}

	e, ok, err := s.repo.get(ctx, Endpoint{ID: endpointID, UserID: userID})
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotFound
	}

	if in.Name != "" {
		if _, dup, err := s.repo.get(ctx, Endpoint{
			ApplicationID: e.ApplicationID,
			TableID:       e.TableID,
			Name:          in.Name,
		}); err != nil {
			return "", err
		} else if dup {
			return "", ErrDuplicateName
		}
	}

	var t *database.Table
	if in.TableID != "" {
		t, err = s.databaseSvc.GetTable(ctx, &database.Table{ID: in.TableID})
		if err != nil {
			return "", err
		}
	} else {
		t, err = s.databaseSvc.GetTable(ctx, &database.Table{ID: e.TableID})
		if err != nil {
			return "", err
		}
	}

	cols, err := s.databaseSvc.ListColumns(ctx, database.Column{TableID: t.ID})
	if err != nil {
		return "", err
	}
	if len(cols) == 0 {
		return "", ErrNoColumns
	}
	if in.OrderBy != "" {
		if err := validateOrderByColumnExist(cols, in.OrderBy); err != nil {
			return "", ErrInvalidOrderBy
		}
	}

	var q string
	if in.Method != "" {
		switch in.Method {
		case "GET":
			if len(in.Columns) == 0 {
				q = fmt.Sprintf("SELECT * FROM %s", t.Name)
			} else {
				q = fmt.Sprintf("SELECT %s FROM %s", strings.Join(in.Columns, ", "), t.Name)
			}
		case "POST":
			vals := make([]string, len(in.Columns))
			for i := range in.Columns {
				vals[i] = fmt.Sprintf("$%d", i+1)
			}
			q = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", t.Name, strings.Join(in.Columns, ", "), strings.Join(vals, ", "))
		}
	}

	if in.Method == "" {
		in.Method = e.Method
	}
	if in.TableID == "" {
		in.TableID = e.TableID
	}
	if in.Name == "" {
		in.Name = e.Name
	}
	if len(in.Columns) == 0 {
		in.Columns = e.Columns
	}

	upd := Endpoint{
		Name:    in.Name,
		TableID: in.TableID,
		Method:  in.Method,
		Columns: in.Columns,
		Query:   q,
	}
	if in.IsPublic != nil && (*in.IsPublic != *e.IsPublic) {
		upd.IsPublic = utils.BoolPointer(*in.IsPublic)
	}

	if err := s.repo.update(ctx, e, upd); err != nil {
		return "", err
	}

	_, _ = s.auditSvc.Create(ctx, audit.LogInput{
		UserID:      userID,
		Action:      "update",
		EntityType:  "endpoint",
		EntityID:    e.ID,
		Description: "updated endpoint",
		Metadata: map[string]any{
			"name":     upd.Name,
			"table_id": upd.TableID,
			"method":   upd.Method,
			"columns":  upd.Columns,
		},
		IPAddress:     utils.GetIPAddressFromCtx(ctx),
		ApplicationID: e.ApplicationID,
	})

	url := fmt.Sprintf("%s/endpoint/execute/%s", utils.GetConfig().ServerBaseURL, e.IdentifierUUID)
	return url, nil
}

func coalesceBoolPtr(p *bool, d bool) bool {
	if p == nil {
		return d
	}
	return *p
}

func openSqlxConnection(dbModel *database.Database) (*sqlx.DB, error) {
	rawPassword, err := utils.Decrypt(dbModel.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return nil, err
	}
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%v password=%s user=%s dbname=%s sslmode=disable", dbModel.Host, dbModel.Port, rawPassword, dbModel.Username, dbModel.Database))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func executeInsertQuery(db *sqlx.DB, query string, values []interface{}) error {
	_, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}
	return nil
}

func executeFetchQuery(db *sqlx.DB, query string, returnColumns []string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var scanColumns []interface{}
	if len(returnColumns) == 0 {
		scanColumns = make([]interface{}, len(columns))
		for i := range columns {
			scanColumns[i] = new(interface{})
		}
	} else {
		scanColumns = make([]interface{}, len(returnColumns))
		for i, col := range returnColumns {
			for _, c := range columns {
				if strings.EqualFold(c, col) {
					scanColumns[i] = new(interface{})
					break
				}
			}
		}
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.Scan(scanColumns...)
		if err != nil {
			return nil, err
		}

		for i, col := range columns {
			if len(returnColumns) == 0 || contains(returnColumns, col) {
				row[col] = *(scanColumns[i].(*interface{}))
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func validateApiKeyInRequest(c *fiber.Ctx, apiKey string) (bool, error) {
	reqHeaderKey := c.Get("X-Apikey")
	if reqHeaderKey == "" || reqHeaderKey != apiKey {
		return false, nil
	}
	return true, nil
}
