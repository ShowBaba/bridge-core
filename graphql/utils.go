package graphql

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
	"gorm.io/gorm"
)

func makeNodeListType(name string, nodeType *graphql.Object) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: name,
		Fields: graphql.Fields{
			"nodes":      &graphql.Field{Type: graphql.NewList(nodeType)},
			"totalCount": &graphql.Field{Type: graphql.Int},
		},
	})
}

func makeListField(listType graphql.Output, resolve graphql.FieldResolveFn) *graphql.Field {
	var (
		listField = &graphql.Field{
			Type:    listType,
			Resolve: resolve,
			Args: graphql.FieldConfigArgument{
				"limit":   &graphql.ArgumentConfig{Type: graphql.Int},
				"offset":  &graphql.ArgumentConfig{Type: graphql.Int},
				"orderBy": &graphql.ArgumentConfig{Type: graphql.String},
			},
		}
		fields graphql.FieldDefinitionMap
	)

	switch listType.Name() {
	case "UserList":
		fields = UserType.Fields()
	case "ApplicationList":
		fields = ApplicationType.Fields()
	case "DatabaseList":
		fields = DatabaseType.Fields()

	case "ColumnList":
		fields = ColumnType.Fields()
	case "IndexType":
		fields = ColumnType.Fields()
	case "EndpointList":
		fields = EndpointType.Fields()
	case "SchemaList":
		fields = SchemaType.Fields()
	case "StreamLogList":
		fields = StreamLogType.Fields()
	case "TableList":
		fields = TableType.Fields()
	case "EndpointScriptList":
		fields = EndpointScriptType.Fields()
	case "DashboardList":
		fields = DashboardType.Fields()
	case "AuditList":
		fields = AuditType.Fields()
	}

	for key, val := range fields {
		if _, ok := listField.Args[key]; !ok {
			listField.Args[key] = &graphql.ArgumentConfig{Type: val.Type}
		}
	}
	return listField
}

func parseDbClause(params graphql.ResolveParams, tx *gorm.DB, nodeType *graphql.Object) *gorm.DB {
	// set all query fields
	if limit, ok := params.Args["limit"].(int); ok {
		tx = tx.Limit(limit)
	}

	if offset, ok := params.Args["offset"].(int); ok {
		tx = tx.Offset(offset)
	}

	if orderBy, ok := params.Args["orderBy"].(string); ok {
		direction := "ASC"
		if strings.Index(orderBy, "-") == 0 {
			direction = "DESC"
			orderBy = orderBy[1:]
		}
		tx = tx.Order(orderBy + " " + direction)
	}

	var fields graphql.FieldDefinitionMap = nodeType.Fields()

	for key, field := range fields {
		val, ok := params.Args[key]
		if !ok {
			continue
		}
		if key == "first_name" && nodeType.Name() == "User" {
			tx = tx.Where("LOWER(first_name) LIKE ?", fmt.Sprintf(`%%%s%%`, strings.ToLower(val.(string))))
			continue
		}
		if key == "last_name" && nodeType.Name() == "User" {
			tx = tx.Where("LOWER(last_name) LIKE ?", fmt.Sprintf(`%%%s%%`, strings.ToLower(val.(string))))
			continue
		}
		if key == "name" && nodeType.Name() == "Application" {
			tx = tx.Where("LOWER(name) LIKE ?", fmt.Sprintf(`%%%s%%`, strings.ToLower(val.(string))))
			continue
		}
		if key == "name" && nodeType.Name() == "Database" {
			tx = tx.Where("LOWER(name) LIKE ? OR LOWER(database) LIKE ?", fmt.Sprintf(`%%%s%%`, strings.ToLower(val.(string))), fmt.Sprintf(`%%%s%%`, strings.ToLower(val.(string))))
			continue
		}
		// handle tables with possible ambiguous id
		if key == "id" && nodeType.Name() == "Application" {
			tx = tx.Where("applications.id = ?", val.(string))
			continue
		}
		if key == "id" && nodeType.Name() == "Database" {
			tx = tx.Where("databases.id = ?", val.(string))
			continue
		}
		if key == "id" && nodeType.Name() == "Column" {
			tx = tx.Where("columns.id = ?", val.(string))
			continue
		}
		if key == "id" && nodeType.Name() == "Endpoint" {
			tx = tx.Where("endpoints.id = ?", val.(string))
			continue
		}
		if key == "id" && nodeType.Name() == "Schema" {
			tx = tx.Where("schemas.id = ?", val.(string))
			continue
		}
		if key == "id" && nodeType.Name() == "Table" {
			tx = tx.Where("tables.id = ?", val.(string))
			continue
		}

		key = underscore(key)
		switch field.Type {
		case graphql.String:
			if match := regexp.MustCompile(`((>|<)=?)\s*(.*?)$`).FindStringSubmatch(val.(string)); len(match) > 0 {
				tx = tx.Where(key+" "+match[1]+"?", match[3])
			} else {
				tx = tx.Where(key+" = ?", val.(string))
			}
		case graphql.Boolean:
			tx = tx.Where(key+" = ?", val.(bool))
		case graphql.Int:

			if intVal, ok := val.(int); ok {
				tx = tx.Where(key+" = ?", intVal)
			} else if strVal, ok := val.(string); ok {
				if intVal, err := strconv.Atoi(strVal); err == nil {
					tx = tx.Where(key+" = ?", intVal)
				} else if match := regexp.MustCompile(`(>=?)\s*(\d+),(<=?)\s*(\d+)`).FindStringSubmatch(strVal); len(match) > 0 {
					intVal, _ = strconv.Atoi(match[2])
					tx = tx.Where(key+" "+match[1]+" ?", intVal)

					intVal, _ = strconv.Atoi(match[4])
					tx = tx.Where(key+" "+match[3]+" ?", intVal)
				} else if match := regexp.MustCompile(`((>|<)=?)\s*(\d+)`).FindStringSubmatch(strVal); len(match) > 0 {
					intVal, _ = strconv.Atoi(match[3])
					tx = tx.Where(key+" "+match[1]+" ?", intVal)
				}
			}
		case graphql.Float:
			// handle >= or <= int value
			if floatVal, ok := val.(float64); ok {
				tx = tx.Where(key+" = ?", floatVal)
			} else if strVal, ok := val.(string); ok {
				if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
					tx = tx.Where(key+" = ?", floatVal)
				} else if match := regexp.MustCompile(`(>=?)\s*(\d+(\.\d+)?),(<=?)\s*(\d+(\.\d+)?)`).FindStringSubmatch(strVal); len(match) > 0 {
					floatVal, _ = strconv.ParseFloat(match[2], 64)
					tx = tx.Where(key+" "+match[1]+" ?", floatVal)

					floatVal, _ = strconv.ParseFloat(match[5], 64)
					tx = tx.Where(key+" "+match[4]+" ?", floatVal)
				} else if match := regexp.MustCompile(`((>|<)=?)\s*(\d+(\.\d+)?)`).FindStringSubmatch(strVal); len(match) > 0 {
					floatVal, _ = strconv.ParseFloat(match[3], 64)
					tx = tx.Where(key+" "+match[1]+" ?", floatVal)
				}
			}
		}
	}

	return tx
}

var camel = regexp.MustCompile("(^[^A-Z0-9]*|[A-Z0-9]*)([A-Z0-9][^A-Z]+|$)")

func underscore(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

func calcDelta(cur, prev int64) (deltaPercent float64, trend string) {
	trend = "up"
	if cur < prev {
		trend = "down"
	}
	if prev == 0 {
		if cur == 0 {
			return 0, "up"
		}
		return 100, "up"
	}
	return (float64(cur-prev) * 100.0) / float64(prev), trend
}

func humanizeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return pluralize(int(d.Minutes()), "min") + " ago"
	}
	if d < 24*time.Hour {
		return pluralize(int(d.Hours()), "hr") + " ago"
	}
	return pluralize(int(d.Hours()/24), "day") + " ago"
}

func pluralize(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

func nullOrStr(s sql.NullString) any {
	if s.Valid {
		return s.String
	}
	return nil
}

func avgOrNil(v sql.NullFloat64) interface{} {
	if v.Valid {
		return math.Round(v.Float64)
	}
	return nil
}
func avgOrZero(v sql.NullFloat64) float64 {
	if v.Valid {
		return math.Round(v.Float64)
	}
	return 0
}
func roundOrZero(v float64) float64 { return math.Round(v) }

func roundIf(v float64) float64 { return math.Round(v) }
func nullFloatToAny(v sql.NullFloat64) interface{} {
	if v.Valid {
		return math.Round(v.Float64)
	}
	return nil
}
