package graphql

import (
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

var UserType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"email":      &graphql.Field{Type: graphql.String},
			"firstname":  &graphql.Field{Type: graphql.String},
			"lastname":   &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{Type: graphql.DateTime},
			"updated_at": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var ApplicationType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Application",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"name":       &graphql.Field{Type: graphql.String},
			"api_key":    &graphql.Field{Type: graphql.String},
			"user_id":    &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{Type: graphql.DateTime},
			"updated_at": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var DatabaseType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Database",
		Fields: graphql.Fields{
			"id":             &graphql.Field{Type: graphql.String},
			"name":           &graphql.Field{Type: graphql.String},
			"host":           &graphql.Field{Type: graphql.String},
			"port":           &graphql.Field{Type: graphql.Int},
			"database":       &graphql.Field{Type: graphql.String},
			"username":       &graphql.Field{Type: graphql.String},
			"password":       &graphql.Field{Type: graphql.String},
			"db_engine":      &graphql.Field{Type: graphql.String},
			"ssl_mode":       &graphql.Field{Type: graphql.String},
			"application_id": &graphql.Field{Type: graphql.String},
			"created_at":     &graphql.Field{Type: graphql.DateTime},
			"updated_at":     &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var ColumnType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Column",
		Fields: graphql.Fields{
			"id":             &graphql.Field{Type: graphql.String},
			"table_id":       &graphql.Field{Type: graphql.String},
			"user_id":        &graphql.Field{Type: graphql.String},
			"name":           &graphql.Field{Type: graphql.String},
			"data_type":      &graphql.Field{Type: graphql.String},
			"is_nullable":    &graphql.Field{Type: graphql.Boolean},
			"default_value":  &graphql.Field{Type: graphql.String},
			"is_primary_key": &graphql.Field{Type: graphql.Boolean},

			"fk_ref_schema": &graphql.Field{Type: graphql.String},
			"fk_ref_table":  &graphql.Field{Type: graphql.String},
			"fk_ref_column": &graphql.Field{Type: graphql.String},
			"fk_constraint": &graphql.Field{Type: graphql.String},

			"created_at": &graphql.Field{Type: graphql.DateTime},
			"updated_at": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var IndexType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Index",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"table_id":   &graphql.Field{Type: graphql.String},
			"user_id":    &graphql.Field{Type: graphql.String},
			"name":       &graphql.Field{Type: graphql.String},
			"definition": &graphql.Field{Type: graphql.String},
			"is_unique":  &graphql.Field{Type: graphql.Boolean},
			"created_at": &graphql.Field{Type: graphql.DateTime},
			"updated_at": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

func parseLiteral(v ast.Value) interface{} {
	switch n := v.(type) {
	case *ast.ObjectValue:
		m := map[string]interface{}{}
		for _, f := range n.Fields {
			m[f.Name.Value] = parseLiteral(f.Value)
		}
		return m
	case *ast.ListValue:
		out := make([]interface{}, len(n.Values))
		for i, it := range n.Values {
			out[i] = parseLiteral(it)
		}
		return out
	case *ast.StringValue:
		return n.Value
	case *ast.IntValue:
		return n.Value
	case *ast.FloatValue:
		return n.Value
	case *ast.BooleanValue:
		return n.Value
	case *ast.EnumValue:
		return n.Value
	default:
		return nil
	}
}

var JSON = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "Arbitrary JSON value",
	Serialize:   func(value interface{}) interface{} { return value },
	ParseValue:  func(value interface{}) interface{} { return value },
	ParseLiteral: func(v ast.Value) interface{} {
		return parseLiteral(v)
	},
})

var EndpointType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Endpoint",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.String},
			"name":            &graphql.Field{Type: graphql.String},
			"application_id":  &graphql.Field{Type: graphql.String},
			"table_id":        &graphql.Field{Type: graphql.String},
			"database_id":     &graphql.Field{Type: graphql.String},
			"user_id":         &graphql.Field{Type: graphql.String},
			"method":          &graphql.Field{Type: graphql.String},
			"path":            &graphql.Field{Type: graphql.String},
			"version":         &graphql.Field{Type: graphql.String},
			"query_template":  &graphql.Field{Type: graphql.String},
			"is_public":       &graphql.Field{Type: graphql.Boolean},
			"columns":         &graphql.Field{Type: graphql.NewList(graphql.String)},
			"limit_default":   &graphql.Field{Type: graphql.Int},
			"limit_max":       &graphql.Field{Type: graphql.Int},
			"order_by":        &graphql.Field{Type: graphql.String},
			"order_direction": &graphql.Field{Type: graphql.String},
			"timeout_ms":      &graphql.Field{Type: graphql.Int},
			"url":             &graphql.Field{Type: graphql.String},
			"param_schema":    &graphql.Field{Type: JSON},
			"query_schema":    &graphql.Field{Type: JSON},
			"body_schema":     &graphql.Field{Type: JSON},
			"created_at":      &graphql.Field{Type: graphql.DateTime},
			"updated_at":      &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var SchemaType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Schema",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.String},
			"database_id": &graphql.Field{Type: graphql.String},
			"name":        &graphql.Field{Type: graphql.String},
			"created_at":  &graphql.Field{Type: graphql.DateTime},
			"updated_at":  &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var StreamLogType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "StreamLog",
		Fields: graphql.Fields{
			"message":     &graphql.Field{Type: graphql.String},
			"level":       &graphql.Field{Type: graphql.String},
			"source":      &graphql.Field{Type: graphql.String},
			"timestamp":   &graphql.Field{Type: graphql.String},
			"application": &graphql.Field{Type: graphql.String},
			"user":        &graphql.Field{Type: graphql.String},
			"created_at":  &graphql.Field{Type: graphql.DateTime},
			"updated_at":  &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var TableType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Table",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.String},
			"schema_id":   &graphql.Field{Type: graphql.String},
			"name":        &graphql.Field{Type: graphql.String},
			"database_id": &graphql.Field{Type: graphql.String},
			"created_at":  &graphql.Field{Type: graphql.DateTime},
			"updated_at":  &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var EndpointScriptType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "EndpointScript",
		Fields: graphql.Fields{
			"id":                &graphql.Field{Type: graphql.String},
			"endpoint_id":       &graphql.Field{Type: graphql.String},
			"kind":              &graphql.Field{Type: graphql.String},
			"code":              &graphql.Field{Type: graphql.String},
			"lang":              &graphql.Field{Type: graphql.String},
			"enabled":           &graphql.Field{Type: graphql.Boolean},
			"script_timeout_ms": &graphql.Field{Type: graphql.Int},
			"created_at":        &graphql.Field{Type: graphql.DateTime},
			"updated_at":        &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var LatencyPointType = graphql.NewObject(graphql.ObjectConfig{
	Name: "LatencyPoint",
	Fields: graphql.Fields{
		"ts":  &graphql.Field{Type: graphql.String}, // RFC3339
		"p50": &graphql.Field{Type: graphql.Float},
		"p95": &graphql.Field{Type: graphql.Float},
		"p99": &graphql.Field{Type: graphql.Float},
	},
})

var DashboardType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Dashboard",
	Fields: graphql.Fields{
		"lastUpdatedISO": &graphql.Field{Type: graphql.String},
		"user": &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: "DashboardUser",
				Fields: graphql.Fields{
					"displayName": &graphql.Field{Type: graphql.String},
					"avatarUrl":   &graphql.Field{Type: graphql.String},
				},
			}),
		},
		"summary": &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: "DashboardSummary",
				Fields: graphql.Fields{
					"applications": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name:   "SummaryApplications",
							Fields: graphql.Fields{"count": &graphql.Field{Type: graphql.Int}},
						}),
					},
					"databases": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name:   "SummaryDatabases",
							Fields: graphql.Fields{"count": &graphql.Field{Type: graphql.Int}},
						}),
					},
					"endpoints": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name:   "SummaryEndpoints",
							Fields: graphql.Fields{"count": &graphql.Field{Type: graphql.Int}},
						}),
					},
					"api_requests": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name: "SummaryAPIRequests",
							Fields: graphql.Fields{
								"count":        &graphql.Field{Type: graphql.Int},
								"deltaPercent": &graphql.Field{Type: graphql.Float},
								"deltaWindow":  &graphql.Field{Type: graphql.String},
								"trend": &graphql.Field{
									Type: graphql.NewEnum(graphql.EnumConfig{
										Name: "APIDeltaTrend",
										Values: graphql.EnumValueConfigMap{
											"up":   &graphql.EnumValueConfig{Value: "up"},
											"down": &graphql.EnumValueConfig{Value: "down"},
										},
									}),
								},
							},
						}),
					},
				},
			}),
		},
		"performance": &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: "DashboardPerformance",
				Fields: graphql.Fields{
					"reliability": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name: "PerformanceReliability",
							Fields: graphql.Fields{
								"successRatePercent": &graphql.Field{Type: graphql.Float},
								"errorRatePercent":   &graphql.Field{Type: graphql.Float},
								"errorBreakdown": &graphql.Field{
									Type: graphql.NewObject(graphql.ObjectConfig{
										Name: "ErrorBreakdown",
										Fields: graphql.Fields{
											"x4xx": &graphql.Field{Type: graphql.Int},
											"x5xx": &graphql.Field{Type: graphql.Int},
										},
									}),
								},
							},
						}),
					},
					"latency": &graphql.Field{
						Type: graphql.NewObject(graphql.ObjectConfig{
							Name: "PerformanceLatency",
							Fields: graphql.Fields{
								"average":       &graphql.Field{Type: graphql.Float},
								"p95":           &graphql.Field{Type: graphql.Float},
								"p99":           &graphql.Field{Type: graphql.Float},
								"unit":          &graphql.Field{Type: graphql.String},
								"window":        &graphql.Field{Type: graphql.String},
								"bucketSizeSec": &graphql.Field{Type: graphql.Int},
								"timeseries":    &graphql.Field{Type: graphql.NewList(LatencyPointType)},
							},
						}),
					},
					"topEndpoints": &graphql.Field{
						Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
							Name: "TopEndpoint",
							Fields: graphql.Fields{
								"path":         &graphql.Field{Type: graphql.String},
								"rpm":          &graphql.Field{Type: graphql.Float},
								"avgLatencyMs": &graphql.Field{Type: graphql.Float},
							},
						})),
					},
					"slowestEndpoints": &graphql.Field{
						Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
							Name: "SlowEndpoint",
							Fields: graphql.Fields{
								"path":         &graphql.Field{Type: graphql.String},
								"avgLatencyMs": &graphql.Field{Type: graphql.Float},
								"p95LatencyMs": &graphql.Field{Type: graphql.Float},
							},
						})),
					},
				},
			}),
		},
		"systemHealth": &graphql.Field{
			Type: graphql.NewObject(graphql.ObjectConfig{
				Name: "SystemHealth",
				Fields: graphql.Fields{
					"apiUptimePercent": &graphql.Field{Type: graphql.Float},
					"averageLatencyMs": &graphql.Field{Type: graphql.Float},
					"errorRatePercent": &graphql.Field{Type: graphql.Float},
				},
			}),
		},
	},
})
var AuditType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Audit",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.String,
			},
			"user_id": &graphql.Field{
				Type: graphql.String,
			},
			"application_id": &graphql.Field{
				Type: graphql.String,
			},
			"action": &graphql.Field{
				Type: graphql.String,
			},
			"entity_type": &graphql.Field{
				Type: graphql.String,
			},
			"entity_id": &graphql.Field{
				Type: graphql.String,
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"metadata": &graphql.Field{
				Type: graphql.String,
			},
			"username": &graphql.Field{
				Type: graphql.String,
			},
			"ip_address": &graphql.Field{
				Type: graphql.String,
			},
			"ts":  &graphql.Field{Type: graphql.String},
			"ago": &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{
				Type: graphql.DateTime,
			},
		},
	},
)
