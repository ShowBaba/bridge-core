package graphql

import (
	"errors"
	"fmt"
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
					tx = parseDbClause(params, tx, ApplicationType).Order("created_at DESC")
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
					tx = parseDbClause(params, tx, DatabaseType).Order("created_at DESC")
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
					tx = parseDbClause(params, tx, ColumnType).Order("created_at DESC")
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
			"index": makeListField(
				makeNodeListType("IndexList", IndexType),
				func(params graphql.ResolveParams) (interface{}, error) {
					userID, ok := params.Context.Value("id").(string)
					if !ok {
						return nil, errors.New("unauthorized access")
					}
					var (
						list    ListResult
						tx      *gorm.DB
						indexes []database.Index
					)
					tx = db.Model(&database.Index{})
					tx = tx.Where("user_id = ?", userID)
					tx = parseDbClause(params, tx, IndexType)
					res := tx.Debug().Scan(&indexes)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range indexes {
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
					tx = parseDbClause(params, tx, EndpointType).Order("created_at DESC")
					res := tx.Debug().Scan(&endpoints)

					if res.Error != nil {
						return nil, res.Error
					}

					if res.RowsAffected > 0 {
						appIDSet := make(map[string]struct{}, len(endpoints))
						for _, e := range endpoints {
							appIDSet[e.ApplicationID] = struct{}{}
						}
						appIDs := make([]string, 0, len(appIDSet))
						for id := range appIDSet {
							appIDs = append(appIDs, id)
						}

						type appRow struct {
							ID   string
							Slug string
						}
						var apps []appRow
						if err := db.Table("applications").
							Select("id, slug").
							Where("id IN ?", appIDs).
							Where("user_id = ?", userID).
							Scan(&apps).Error; err != nil {
							return nil, err
						}
						slugByApp := make(map[string]string, len(apps))
						for _, a := range apps {
							slugByApp[a.ID] = a.Slug
						}

						base := utils.GetConfig().ServerBaseURL
						list.Nodes = make([]interface{}, 0, len(endpoints))
						for _, u := range endpoints {
							slug := slugByApp[u.ApplicationID]
							version := "v1"
							if u.Version != "" {
								version = u.Version
							}
							u.URL = utils.FormatPublicURL(base, version, slug, u.Path)
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
					tx = parseDbClause(params, tx, AuditType).Order("created_at DESC")
					res := tx.Debug().Scan(&audits)
					if res.RowsAffected > 0 {
						list.Nodes = []interface{}{}
						for _, u := range audits {
							row := db.Raw(`SELECT 
	    		  	COALESCE(first_name, '')      AS first_name
							FROM users WHERE id = ?`, userID).Row()
							if err := row.Scan(&u.Username); err != nil {
								fmt.Println("Error scanning row:", err)
								return nil, errors.New("something went wrong")
							}

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
