package database

import (
	"database/sql"
	"fmt"
	"github.com/showbaba/query-bridge/bridge-core/models"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func FetchDatabaseInfo(db *sql.DB, sqlLogCh chan<- string, result chan<- utils.SchemaData, errCh chan<- error) {
	var err error

	schemas, err := FetchSchemas(db, sqlLogCh)
	if err != nil {
		errCh <- fmt.Errorf("failed to fetch schemas: %v", err)
		return
	}
	// var tables []string
	for _, schema := range schemas {
		tables, err := FetchTables(db, schema, sqlLogCh)
		if err != nil {
			errCh <- err
			continue
		}
		var tableData []utils.TableData
		for _, table := range tables {
			columns, err := FetchColumns(db, schema, table, sqlLogCh)
			if err != nil {
				errCh <- err
				continue
			}
			tableData = append(tableData, utils.TableData{Table: table, Columns: columns})
		}
		result <- utils.SchemaData{Schema: schema, Tables: tableData}
	}
}

func FetchSchemas(db *sql.DB, sqlLogCh chan<- string) ([]string, error) {
	var schemas []string

	query := "SELECT schema_name FROM information_schema.schemata WHERE schema_name NOT LIKE 'pg_%' AND schema_name != 'information_schema'"
	rows, err := db.Query(query)
	sqlLogCh <- query
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var schema string
		err := rows.Scan(&schema)
		if err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

func FetchTables(db *sql.DB, schema string, sqlLogCh chan<- string) ([]string, error) {
	var tables []string

	query := fmt.Sprintf("SELECT table_name FROM information_schema.tables WHERE table_schema = '%s'", schema)
	sqlLogCh <- query
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		err := rows.Scan(&table)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func FetchColumns(db *sql.DB, schema, tableName string, sqlLogCh chan<- string) ([]string, error) {
	var columns []string

	query := fmt.Sprintf(`SELECT column_name FROM information_schema.columns WHERE table_schema = '%s' AND table_name = '%s' AND column_name NOT LIKE 'pg_%%' AND column_name NOT LIKE 'sys_%%'`, schema, tableName)
	sqlLogCh <- query

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var column string
		err := rows.Scan(&column)
		if err != nil {
			return nil, err
		}
		columns = append(columns, column)
	}

	return columns, nil
}

func LogSqlQuery(mongoClient *mongo.Client, applicationID uint, userID uint, ch <-chan string, errCh chan<- error) {
	for logData := range ch {
		var stream *models.StreamLog
		err := stream.Insert(ctx, mongoClient, logData, applicationID, userID)
		if err != nil {
			errCh <- err
		}
	}
}

func StoreData(db *gorm.DB, dbID, userID uint, ch <-chan utils.SchemaData, errCh chan<- error) {
	for data := range ch {
		schema := &models.Schema{
			DatabaseID: dbID,
			Name:       data.Schema,
			UserID:     userID,
		}
		var schemaID uint
		existingSchema, exist, err := schema.FetchSchemaByNameAndDatabaseID(db)
		if err != nil {
			errCh <- err
			continue
		}
		if exist {
			err = schema.Update(db, *schema)
			if err != nil {
				errCh <- err
				continue
			}
			schemaID = existingSchema.ID
		} else {
			schemaID, err = schema.Insert(db)
			if err != nil {
				errCh <- err
				continue
			}
		}
		for _, tableData := range data.Tables {
			table := &models.Table{
				SchemaID:   schemaID,
				Name:       tableData.Table,
				DatabaseID: dbID,
				UserID:     userID,
			}
			var tableID uint
			existingTable, exist, err := table.FetchTableByNameAndSchemaID(db)
			if err != nil {
				errCh <- err
				continue
			}
			if exist {
				err = table.Update(db, *table)
				if err != nil {
					errCh <- err
					continue
				}
				tableID = existingTable.ID
			} else {
				tableID, err = table.Insert(db)
				if err != nil {
					errCh <- err
				}
			}

			for _, column := range tableData.Columns {
				column := &models.Column{
					TableID: tableID,
					Name:    column,
					UserID:  userID,
				}
				_, exist, err := column.FetchColumnByNameAndTableID(db)
				if err != nil {
					errCh <- err
					continue
				}
				if exist {
					err = column.Update(db, *column)
					if err != nil {
						errCh <- err
						continue
					}
				} else {
					if err := column.Insert(db); err != nil {
						errCh <- err
					}
				}
			}
		}
	}
}
