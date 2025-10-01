package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jmoiron/sqlx"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"github.com/xeipuuv/gojsonschema"
)

func coalesceStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func coalesceBoolPtr(p *bool, d bool) bool {
	if p == nil {
		return d
	}
	return *p
}
func toJSON(v any) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}

func matchPath(pattern string, actual string) (map[string]string, bool) {
	pSeg := strings.Split(strings.Trim(pattern, "/"), "/")
	aSeg := strings.Split(strings.Trim(actual, "/"), "/")
	if len(pSeg) != len(aSeg) {
		return nil, false
	}
	out := map[string]string{}
	for i := range pSeg {
		if strings.HasPrefix(pSeg[i], ":") {
			name := strings.TrimPrefix(pSeg[i], ":")
			out[name] = aSeg[i]
			continue
		}
		if pSeg[i] != aSeg[i] {
			return nil, false
		}
	}
	return out, true
}

var paramRe = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

func bindNamed(
	sqlTmpl string,
	pathParams map[string]string,
	queryParams map[string][]string,
	bodyNamed map[string]any,
	bodyPositional []interface{},
) (string, []interface{}, error) {
	type source func(string) (any, bool)
	lookups := []source{
		func(k string) (any, bool) { v, ok := pathParams[k]; return v, ok },
		func(k string) (any, bool) {
			if vs, ok := queryParams[k]; ok && len(vs) > 0 {
				return vs[0], true
			}
			return nil, false
		},
		func(k string) (any, bool) {
			v, ok := bodyNamed[k]
			return v, ok
		},
	}

	var args []interface{}
	i := 1
	sqlOut := paramRe.ReplaceAllStringFunc(sqlTmpl, func(m string) string {
		key := strings.TrimPrefix(m, ":")
		for _, lk := range lookups {
			if v, ok := lk(key); ok {
				args = append(args, v)
				s := fmt.Sprintf("$%d", i)
				i++
				return s
			}
		}
		if len(bodyPositional) > 0 {
			args = append(args, bodyPositional[0])
			bodyPositional = bodyPositional[1:]
		} else {
			args = append(args, nil)
		}
		s := fmt.Sprintf("$%d", i)
		i++
		return s
	})
	return sqlOut, args, nil
}

func openSqlxConnection(dbModel *database.Database) (*sqlx.DB, error) {
	rawPassword, err := utils.Decrypt(dbModel.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf(
		"host=%s port=%v password=%s user=%s dbname=%s sslmode=%s",
		dbModel.Host, dbModel.Port, rawPassword, dbModel.Username, dbModel.Database,
		coalesceStr(dbModel.SSLMode, "disable"),
	)
	return sqlx.Connect("postgres", dsn)
}

func executeInsertQuery(db *sqlx.DB, query string, args []interface{}) error {
	_, err := db.Exec(query, args...)
	return err
}

func executeFetchQuery(db *sqlx.DB, query string, _ []string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		raw := make([]interface{}, len(cols))
		dest := make([]interface{}, len(cols))
		for i := range raw {
			dest[i] = &raw[i]
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		row := map[string]interface{}{}
		for i, c := range cols {
			row[c] = raw[i]
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

type endpointSpec interface {
	GetMethod() string
	GetColumns() []string
	GetOrderBy() string
	GetOrderDirection() string
	GetLimitDefault() *uint
}

func buildQueryTemplate(table string, in endpointSpec) string {
	switch strings.ToUpper(in.GetMethod()) {
	case "GET":
		cols := "*"
		if cs := in.GetColumns(); len(cs) > 0 {
			cols = strings.Join(cs, ", ")
		}
		q := "SELECT " + cols + " FROM " + table
		if ob := in.GetOrderBy(); ob != "" {
			q += " ORDER BY " + ob
			if od := in.GetOrderDirection(); od != "" {
				q += " " + od
			}
		}
		if ld := in.GetLimitDefault(); ld != nil && *ld > 0 {
			q += fmt.Sprintf(" LIMIT %d", *ld)
		}
		return q

	case "POST":
		cs := in.GetColumns()
		if len(cs) == 0 {
			return "/* POST requires columns. Provide Columns or write a Query Template. */"
		}
		vals := make([]string, len(cs))
		for i, c := range cs {
			vals[i] = ":" + c
		}
		return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(cs, ", "), strings.Join(vals, ", "))

	case "PUT", "PATCH":
		cs := in.GetColumns()
		if len(cs) == 0 {
			return "/* PUT/PATCH requires columns to update. Provide Columns or write a Query Template. */"
		}
		assigns := make([]string, len(cs))
		for i, c := range cs {
			assigns[i] = fmt.Sprintf("%s = :%s", c, c)
		}
		return fmt.Sprintf("UPDATE %s SET %s WHERE id = :id", table, strings.Join(assigns, ", "))

	case "DELETE":
		return fmt.Sprintf("DELETE FROM %s WHERE id = :id", table)

	default:
		return "/* Unsupported method in simple mode. Provide a Query Template. */"
	}
}

func validateApiKeyInRequest(c *fiber.Ctx, expected string) (bool, error) {
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return false, nil
	}
	if apiKey == expected {
		return true, nil
	}
	return false, nil
}

var (
	spaceRe      = regexp.MustCompile(`\s+`)
	multiSlashRe = regexp.MustCompile(`/+`)
	queryFragRe  = regexp.MustCompile(`[?#].*$`)
	validVersion = regexp.MustCompile(`^v[0-9]+$`)
)

func sanitizeCreateInput(in CreateEndpointInput) CreateEndpointInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Method = strings.ToUpper(strings.TrimSpace(in.Method))
	in.OrderDirection = strings.ToUpper(strings.TrimSpace(in.OrderDirection))
	in.Path = normalizePath(in.Path)
	in.Version = normalizeVersion(in.Version)
	in.Columns = sanitizeColumns(in.Columns)
	in.OrderBy = strings.TrimSpace(in.OrderBy)
	in.QueryTemplate = strings.TrimSpace(in.QueryTemplate)
	return in
}

func sanitizeUpdateInput(in UpdateEndpointInput) UpdateEndpointInput {
	in.Name = strings.TrimSpace(in.Name)
	if in.Method != "" {
		in.Method = strings.ToUpper(strings.TrimSpace(in.Method))
	}
	if in.OrderDirection != "" {
		in.OrderDirection = strings.ToUpper(strings.TrimSpace(in.OrderDirection))
	}
	if in.Path != "" {
		in.Path = normalizePath(in.Path)
	}
	in.Columns = sanitizeColumns(in.Columns)
	in.OrderBy = strings.TrimSpace(in.OrderBy)
	in.QueryTemplate = strings.TrimSpace(in.QueryTemplate)
	return in
}

// Ensures a clean, absolute, slash-normalized path (e.g. "orders/:id " => "/orders/:id")
func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	p = queryFragRe.ReplaceAllString(p, "")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = multiSlashRe.ReplaceAllString(p, "/")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	p = spaceRe.ReplaceAllString(p, "")
	return p
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "v1"
	}
	v = strings.ToLower(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !validVersion.MatchString(v) {
		return "v1"
	}
	return v
}

func sanitizeColumns(cols []string) []string {
	if len(cols) == 0 {
		return cols
	}
	seen := make(map[string]struct{}, len(cols))
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

type ColMeta struct {
	Name       string
	DataType   string
	Nullable   bool
	HasDefault bool
	IsPK       bool
	IsAutoInc  bool
}

type schemaInfo struct {
	Props    map[string]struct{}
	Required map[string]struct{}
}

func parseJSONSchema(raw []byte) schemaInfo {
	out := schemaInfo{Props: map[string]struct{}{}, Required: map[string]struct{}{}}
	if len(raw) == 0 {
		return out
	}
	var s struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	_ = json.Unmarshal(raw, &s)
	for k := range s.Properties {
		out.Props[strings.ToLower(k)] = struct{}{}
	}
	for _, r := range s.Required {
		out.Required[strings.ToLower(r)] = struct{}{}
	}
	return out
}

type defValidationOpts struct {
	Method        string
	TableMeta     map[string]ColMeta
	Columns       []string
	OrderBy       string
	BodySchema    []byte
	QuerySchema   []byte
	ParamSchema   []byte
	QueryTemplate string
}

type defError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e defError) ToFieldError() utils.FieldError {
	return utils.FieldError{Field: e.Field, Message: e.Message}
}

var namedRe = regexp.MustCompile(`:([A-Za-z_][A-Za-z0-9_]*)`)

func extractPlaceholders(sql string) []string {
	m := namedRe.FindAllStringSubmatch(sql, -1)
	out := make([]string, 0, len(m))
	seen := map[string]struct{}{}
	for _, g := range m {
		if len(g) > 1 {
			k := strings.ToLower(g[1])
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				out = append(out, k)
			}
		}
	}
	return out
}

func validateEndpointDefinition(opts defValidationOpts) []defError {
	var errs []defError

	if opts.OrderBy != "" {
		if _, ok := opts.TableMeta[strings.ToLower(opts.OrderBy)]; !ok {
			errs = append(errs, defError{Field: "order_by", Message: "column does not exist"})
		}
	}

	for _, c := range opts.Columns {
		if _, ok := opts.TableMeta[strings.ToLower(c)]; !ok {
			errs = append(errs, defError{Field: "columns", Message: fmt.Sprintf("unknown column %q", c)})
		}
	}

	body := parseJSONSchema(opts.BodySchema)
	query := parseJSONSchema(opts.QuerySchema)
	param := parseJSONSchema(opts.ParamSchema)

	// write methods: ensure required body fields are real & writable
	switch strings.ToUpper(opts.Method) {
	case "POST", "PUT", "PATCH":
		for req := range body.Required {
			meta, ok := opts.TableMeta[req]
			if !ok {
				errs = append(errs, defError{Field: "body_schema.required", Message: fmt.Sprintf("required field %q is not a table column", req)})
				continue
			}
			if meta.IsPK && meta.IsAutoInc {
				errs = append(errs, defError{Field: req, Message: "cannot require auto-generated primary key"})
			}
		}
		// NOT NULL w/o default should be provided
		for _, m := range opts.TableMeta {
			key := strings.ToLower(m.Name)
			if !m.Nullable && !m.HasDefault && !(m.IsPK && m.IsAutoInc) {
				if _, present := body.Required[key]; !present {
					errs = append(errs, defError{
						Field:   key,
						Message: "NOT NULL column without default must be provided (add to body_schema.required or supply via path/query)",
					})
				}
			}
		}
	}

	// If template provided, all :placeholders must be resolvable
	if strings.TrimSpace(opts.QueryTemplate) != "" {
		ph := extractPlaceholders(opts.QueryTemplate)
		allowed := map[string]struct{}{}
		for k := range param.Props {
			allowed[k] = struct{}{}
		}
		for k := range query.Props {
			allowed[k] = struct{}{}
		}
		for k := range body.Props {
			allowed[k] = struct{}{}
		}
		// Also allow table columns (common for write templates)
		for k := range opts.TableMeta {
			allowed[k] = struct{}{}
		}
		for _, n := range ph {
			if _, ok := allowed[n]; !ok {
				errs = append(errs, defError{
					Field:   "query_template",
					Message: fmt.Sprintf("placeholder :%s not defined in path/query/body schema nor as a table column", n),
				})
			}
		}
		// For write methods ensure every required body field is referenced
		if strings.EqualFold(opts.Method, "POST") || strings.EqualFold(opts.Method, "PUT") || strings.EqualFold(opts.Method, "PATCH") {
			has := map[string]struct{}{}
			for _, n := range ph {
				has[n] = struct{}{}
			}
			for req := range body.Required {
				if _, ok := has[req]; !ok {
					errs = append(errs, defError{
						Field:   "query_template",
						Message: fmt.Sprintf("required body field %q not used in SQL template", req),
					})
				}
			}
		}
	}

	return errs
}

func sanitizePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = regexp.MustCompile(`/+`).ReplaceAllString(p, `/`)
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	if strings.ContainsAny(p, " \t\n\r") {
		return "", fmt.Errorf("path contains whitespace")
	}
	return p, nil
}

func validateJSON(schemaRaw []byte, value any) ([]string, error) {
	if len(schemaRaw) == 0 || value == nil {
		return nil, nil
	}
	loader := gojsonschema.NewGoLoader(value)
	schemaLoader := gojsonschema.NewBytesLoader(schemaRaw)
	res, err := gojsonschema.Validate(schemaLoader, loader)
	if err != nil {
		return nil, err
	}
	if res.Valid() {
		return nil, nil
	}
	var msgs []string
	for _, e := range res.Errors() {
		msgs = append(msgs, e.String())
	}
	return msgs, nil
}

func validateJSONValue(schemaRaw []byte, value any) error {
	msgs, err := validateJSON(schemaRaw, value)
	if err != nil {
		return err
	}
	if len(msgs) > 0 {
		return errors.New(strings.Join(msgs, "; "))
	}
	return nil
}

func validateJSONBody(schemaRaw []byte, body any) error {
	return validateJSONValue(schemaRaw, body)
}

func validateJSONParams(schemaRaw []byte, params map[string]string) error {
	if len(schemaRaw) == 0 {
		return nil
	}
	m := make(map[string]any, len(params))
	for k, v := range params {
		m[k] = v
	}
	return validateJSONValue(schemaRaw, m)
}

func validateJSONQuery(schemaRaw []byte, q map[string][]string) error {
	if len(schemaRaw) == 0 {
		return nil
	}
	m := make(map[string]any, len(q))
	for k, vs := range q {
		if len(vs) == 1 {
			m[k] = vs[0]
		} else {
			m[k] = vs
		}
	}
	return validateJSONValue(schemaRaw, m)
}
