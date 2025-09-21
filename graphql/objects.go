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
			"id":            &graphql.Field{Type: graphql.String},
			"Name":          &graphql.Field{Type: graphql.String},
			"Host":          &graphql.Field{Type: graphql.String},
			"Port":          &graphql.Field{Type: graphql.Int},
			"Database":      &graphql.Field{Type: graphql.String},
			"Username":      &graphql.Field{Type: graphql.String},
			"Password":      &graphql.Field{Type: graphql.String},
			"DbEngine":      &graphql.Field{Type: graphql.String},
			"ApplicationID": &graphql.Field{Type: graphql.String},
			"CreatedAt":     &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt":     &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var ColumnType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Column",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.String},
			"TableID":   &graphql.Field{Type: graphql.String},
			"Name":      &graphql.Field{Type: graphql.String},
			"CreatedAt": &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var EndpointType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Endpoint",
		Fields: graphql.Fields{
			"id":             &graphql.Field{Type: graphql.Int},
			"Name":           &graphql.Field{Type: graphql.String},
			"ApplicationID":  &graphql.Field{Type: graphql.String},
			"DatabaseID":     &graphql.Field{Type: graphql.String},
			"TableID":        &graphql.Field{Type: graphql.String},
			"Limit":          &graphql.Field{Type: graphql.Int},
			"OrderBy":        &graphql.Field{Type: graphql.String},
			"OrderDirection": &graphql.Field{Type: graphql.String},
			"Query":          &graphql.Field{Type: graphql.String},
			"IsPublic":       &graphql.Field{Type: graphql.Boolean},
			"CreatedAt":      &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt":      &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var SchemaType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Schema",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"DatabaseID": &graphql.Field{Type: graphql.String},
			"Name":       &graphql.Field{Type: graphql.String},
			"CreatedAt":  &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt":  &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var StreamLogType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "StreamLog",
		Fields: graphql.Fields{
			"Message":   &graphql.Field{Type: graphql.String},
			"Level":     &graphql.Field{Type: graphql.String},
			"Source":    &graphql.Field{Type: graphql.String},
			"Timestamp": &graphql.Field{Type: graphql.String},
			"CreatedAt": &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt": &graphql.Field{Type: graphql.DateTime},
		},
	},
)

var TableType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Table",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"SchemaID":   &graphql.Field{Type: graphql.String},
			"Name":       &graphql.Field{Type: graphql.String},
			"DatabaseID": &graphql.Field{Type: graphql.String},
			"CreatedAt":  &graphql.Field{Type: graphql.DateTime},
			"UpdatedAt":  &graphql.Field{Type: graphql.DateTime},
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
