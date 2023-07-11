package gql

import (
	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge/models"
	"gorm.io/gorm"
)

func Init(db *gorm.DB) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"users": makeListField(
				makeNodeListType("UserList", UserType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list  ListResult
						tx    *gorm.DB
						users []models.User
					)
					tx = parseDbClause(params, db.Model(&models.User{}), UserType)
					res := tx.Debug().Scan(&users)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range users {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"applications": makeListField(
				makeNodeListType("ApplicationList", ApplicationType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list         ListResult
						tx           *gorm.DB
						applications []models.Application
					)
					tx = db.Model(&models.Application{}).Joins("JOIN users ON applications.user_id = users.id")
					tx = parseDbClause(params, tx, ApplicationType)
					res := tx.Debug().Scan(&applications)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range applications {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"databases": makeListField(
				makeNodeListType("DatabaseList", DatabaseType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list      ListResult
						tx        *gorm.DB
						databases []models.Database
					)
					tx = db.Model(&models.Database{}).Joins("JOIN applications ON databases.application_id = applications.id")
					tx = parseDbClause(params, tx, DatabaseType)
					res := tx.Debug().Scan(&databases)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range databases {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"columns": makeListField(
				makeNodeListType("ColumnList", ColumnType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list    ListResult
						tx      *gorm.DB
						columns []models.Column
					)
					tx = db.Model(&models.Column{}).Joins("JOIN tables ON columns.table_id = tables.id")
					tx = parseDbClause(params, tx, ColumnType)
					res := tx.Debug().Scan(&columns)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range columns {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"endpoints": makeListField(
				makeNodeListType("EndpointList", EndpointType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list      ListResult
						tx        *gorm.DB
						endpoints []models.Endpoint
					)
					tx = db.Model(&models.Endpoint{}).Joins("JOIN tables ON endpoints.table_id = tables.id").
						Joins("JOIN applications ON endpoints.table_id = applications.id")
					tx = parseDbClause(params, tx, EndpointType)
					res := tx.Debug().Scan(&endpoints)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range endpoints {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"schemas": makeListField(
				makeNodeListType("SchemaList", SchemaType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list    ListResult
						tx      *gorm.DB
						schemas []models.Schema
					)
					tx = db.Model(&models.Schema{}).Joins("JOIN databases ON schemas.database_id = databases.id")
					tx = parseDbClause(params, tx, SchemaType)
					res := tx.Debug().Scan(&schemas)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range schemas {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			"tables": makeListField(
				makeNodeListType("TableList", TableType),
				func(params graphql.ResolveParams) (interface{}, error) {
					var (
						list   ListResult
						tx     *gorm.DB
						tables []models.Table
					)
					tx = db.Model(&models.Table{}).Joins("JOIN schemas ON tables.schema_id = schemas.id")
					tx = parseDbClause(params, tx, TableType)
					res := tx.Debug().Scan(&tables)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range tables {
							list.Nodes = append(list.Nodes, interface{}(u))
						}
						list.TotalCount = len(list.Nodes)
					}
					return list, nil
				},
			),
			// "stream_logs": makeListField(
				// TODO: implement fetch from mongodb
				// makeNodeListType("StreamLogList", StreamLogType),
				// func(params graphql.ResolveParams) (interface{}, error) {
				// 	var (
				// 		list       ListResult
				// 		tx         *gorm.DB
				// 		streamLogs []models.StreamLog
				// 	)
				// 	tx = db.Model(&models.StreamLog{})
				// 	tx = parseDbClause(params, tx, StreamLogType)
				// 	res := tx.Debug().Scan(&streamLogs)
				// 	if res.RowsAffected > 0 {
				// 		list.Nodes = []interface{}{}
				// 		for _, u := range streamLogs {
				// 			list.Nodes = append(list.Nodes, interface{}(u))
				// 		}
				// 		list.TotalCount = len(list.Nodes)
				// 	}
				// 	return list, nil
				// },
			// ),
		}})
}
