package graphql

import (
	"errors"
	"log"

	"github.com/graphql-go/graphql"
	"github.com/showbaba/query-bridge/bridge-core/application"
	"github.com/showbaba/query-bridge/bridge-core/audit"
	"github.com/showbaba/query-bridge/bridge-core/database"
	"github.com/showbaba/query-bridge/bridge-core/endpoint"
	"github.com/showbaba/query-bridge/bridge-core/user"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

func Init(db *gorm.DB) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"users": makeListField(
				makeNodeListType("UserList", UserType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list  ListResult
						tx    *gorm.DB
						users []user.User
					)
					tx = parseDbClause(params, db.Model(&user.User{}), UserType)
					tx = tx.Where("id = ?", userID)
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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list         ListResult
						tx           *gorm.DB
						applications []application.Application
					)
					tx = db.Model(&application.Application{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, ApplicationType)
					res := tx.Debug().Scan(&applications)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range applications {
							if u.ApiKey != "" {
								// decrypt
								rawApiKey, err := utils.Decrypt(u.ApiKey, []byte(utils.GetConfig().EncryptionKey))
								if err != nil {
									log.Fatal(err)
								}
								u.ApiKey = string(rawApiKey)
							}
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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list      ListResult
						tx        *gorm.DB
						databases []database.Database
					)
					tx = db.Model(&database.Database{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, DatabaseType)
					res := tx.Debug().Scan(&databases)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range databases {
							// decrypt db passwords
							dec, _ := utils.Decrypt(u.Password, []byte(utils.GetConfig().EncryptionKey))
							u.Password = string(dec)
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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						columns []database.Column
					)
					tx = db.Model(&database.Column{})
					tx = tx.Where("user_id = ?", userID)
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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list      ListResult
						tx        *gorm.DB
						endpoints []endpoint.Endpoint
					)
					tx = db.Model(&endpoint.Endpoint{})
					tx = tx.Where("user_id = ?", userID)
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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						schemas []database.Schema
					)
					tx = db.Model(&database.Schema{})
					tx = tx.Where("user_id = ?", userID)

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
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list   ListResult
						tx     *gorm.DB
						tables []database.Table
					)
					tx = db.Model(&database.Table{})
					tx = tx.Where("user_id = ?", userID)
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
			"audits": makeListField( // todo: make application_id required instead
				makeNodeListType("AuditList", AuditType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list   ListResult
						tx     *gorm.DB
						audits []audit.Audit
					)
					tx = db.Model(&audit.Audit{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, AuditType)
					res := tx.Debug().Scan(&audits)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range audits {
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
