package graphql

import (
	"github.com/graphql-go/graphql"
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

var EndpointType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Endpoint",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.String},
			"name":            &graphql.Field{Type: graphql.String},
			"application_id":  &graphql.Field{Type: graphql.String},
			"database_id":     &graphql.Field{Type: graphql.String},
			"table_id":        &graphql.Field{Type: graphql.String},
			"limit":           &graphql.Field{Type: graphql.Int},
			"order_by":        &graphql.Field{Type: graphql.String},
			"order_direction": &graphql.Field{Type: graphql.String},
			"query":           &graphql.Field{Type: graphql.String},
			"is_public":       &graphql.Field{Type: graphql.Boolean},
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
			"message":    &graphql.Field{Type: graphql.String},
			"level":      &graphql.Field{Type: graphql.String},
			"source":     &graphql.Field{Type: graphql.String},
			"timestamp":  &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{Type: graphql.DateTime},
			"updated_at": &graphql.Field{Type: graphql.DateTime},
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
			"ip_address": &graphql.Field{
				Type: graphql.String,
			},
			"created_at": &graphql.Field{
				Type: graphql.DateTime,
			},
		},
	},
)
