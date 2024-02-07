package queues

import (
	"context"
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
		if !isDefaultTable(table) {
			tables = append(tables, table)
		}
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

func LogSqlQuery(ctx context.Context, mongoClient *mongo.Client, applicationID uint, userID uint, ch <-chan string, errCh chan<- error) {
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

var defaultTables = []string{"Overview", "pg_aggregate", "pg_am", "pg_amop", "pg_amproc", "pg_attrdef", "pg_attribute", "pg_authid",
	"pg_auth_members", "pg_cast", "pg_class",
	"pg_collation", "pg_constraint", "pg_conversion", "pg_database", "pg_db_role_setting", "pg_default_acl", "pg_depend",
	"pg_description", "pg_enum", "pg_event_trigger", "pg_extension", "pg_foreign_server", "pg_foreign_data_wrapper", "pg_foreign_table", "pg_index", "pg_inherits",
	"pg_init_privs", "pg_language", "pg_largeobject", "pg_largeobject_metadata", "pg_namespace", "pg_opclass", "pg_operator",
	"pg_opfamily", "pg_parameter_acl", "pg_partitioned_table", "pg_policy", "pg_proc", "pg_publication", "pg_publication_namespace",
	"pg_publication_rel", "pg_range", "pg_replication_origin", "pg_rewrite", "pg_seclabel", "pg_sequence", "pg_shdepend", "pg_shdescription",
	"pg_shseclabel", "pg_statistic", "pg_statistic_ext", "pg_statistic_ext_data", "pg_subscription", "pg_subscription_rel", "pg_tablespace", "pg_transform",
	"pg_trigger", "pg_ts_config", "pg_ts_config_map", "pg_ts_dict", "pg_ts_parser", "pg_ts_template", "pg_type", "pg_user_mapping",
	"pg_stat_activity", "pg_stat_database",
	"pg_stat_user_tables", "pg_stat_user_indexes", "pg_stat_user_functions", "pg_stat_replication", "pg_stat_bgwriter", "pg_stat_ssl", "pg_stat_progress_vacuum",
	"pg_stat_wal_receiver", "pg_stat_subscription", "pg_stat_all_tables", "pg_stat_sys_tables", "pg_stat_sys_indexes", "pg_stat_sys_functions", "pg_stat_xact_all_tables",
	"pg_stat_xact_sys_tables", "pg_stat_xact_user_tables", "pg_stat_all_indexes", "pg_stat_xact_all_indexes", "pg_stat_database_conflicts",
	"pg_settings", "pg_locks", "pg_prepared_xacts",
	"pg_prepared_statements", "pg_cursors", "pg_tables", "pg_views", "pg_indexes", "pg_user_mappings", "pg_user", "pg_group", "pg_roles", "pg_shadow", "pg_auth_members",
	"pg_description", "pg_shdescription", "pg_db_role_setting", "pg_init_privs", "pg_seclabels", "pg_shseclabels", "pg_timezone_abbrevs", "pg_timezone_names", "pg_statistic",
	"pg_subscription_rel", "pg_replication_slots", "pg_publication", "pg_publication_rel", "pg_stat_statements",
}

func isDefaultTable(tableName string) bool {
	for _, table := range defaultTables {
		if tableName == table {
			return true
		}
	}
	return false
}
