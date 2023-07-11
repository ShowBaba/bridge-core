package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

func FetchSchemaTables(db *sql.DB, sqlLogCh chan<- string, result chan<- utils.SchemaData, errCh chan<- error) {
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

	query := fmt.Sprintf(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = '%s' AND table_name = '%s'
	`, schema, tableName)
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

func LogSqlQuery(mongoClient *mongo.Client, ch <-chan string, errCh chan<- error) {
	for logData := range ch {
		collection := mongoClient.Database(utils.QUERY_BRIDGE_MONGO_DB_NAME).Collection("logs")
		currentTime := time.Now()
		_, err := collection.InsertOne(ctx, models.StreamLog{
			Message:   logData,
			Source:    "DATABASE",
			Level:     "INFO",
			Timestamp: currentTime.Format("2006-01-02 15:04:05"),
			CreatedAt: currentTime,
			UpdatedAt: currentTime,
		})
		if err != nil {
			errCh <- err
		}
		// TODO: send log to websocket
	}
}

func StoreData(db *gorm.DB, dbID uint, ch <-chan utils.SchemaData, errCh chan<- error) {
	for data := range ch {
		schema := &models.Schema{
			DatabaseID: dbID,
			Name:       data.Schema,
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
				SchemaID: schemaID,
				Name:     tableData.Table,
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
