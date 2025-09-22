package queues

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/showbaba/query-bridge/bridge-core/utils"
	"gorm.io/gorm"
)

// queues/functions.go
func fetchDatabaseInfo(db *sql.DB, sqlLogCh chan<- string, result chan<- utils.SchemaData, errCh chan<- error) {
	schemas, err := fetchSchemas(db, sqlLogCh)
	if err != nil {
		errCh <- fmt.Errorf("failed to fetch schemas: %v", err)
		return
	}

	for _, schema := range schemas {
		tables, err := fetchTables(db, schema, sqlLogCh)
		if err != nil {
			errCh <- err
			continue
		}

		var tableData []utils.TableData
		for _, table := range tables {
			cols, pks, fks, idxs, err := fetchTableMetadata(db, schema, table, sqlLogCh)
			if err != nil {
				errCh <- err
				continue
			}

			colOut := make([]utils.ColumnMeta, 0, len(cols))
			for _, c := range cols {
				var (
					fkRefSchema *string
					fkRefTable  *string
					fkRefColumn *string
					fkConstr    *string
				)
				if fk, ok := fks[c.Name]; ok {
					fkRefSchema = &fk.RefSchema
					fkRefTable = &fk.RefTable
					fkRefColumn = &fk.RefColumn
					fkConstr = &fk.Constraint
				}
				colOut = append(colOut, utils.ColumnMeta{
					Name:         c.Name,
					DataType:     c.DataType,
					IsNullable:   c.IsNullable,
					DefaultValue: c.DefaultValue,
					IsPrimaryKey: pks[c.Name],
					FKRefSchema:  fkRefSchema,
					FKRefTable:   fkRefTable,
					FKRefColumn:  fkRefColumn,
					FKConstraint: fkConstr,
				})
			}

			idxOut := make([]utils.IndexMeta, 0, len(idxs))
			for _, ix := range idxs {
				idxOut = append(idxOut, utils.IndexMeta{
					Name:       ix.Name,
					Definition: ix.Definition,
					IsUnique:   ix.IsUnique,
				})
			}

			tableData = append(tableData, utils.TableData{
				Table:   table,
				Columns: colOut,
				Indexes: idxOut,
			})
		}

		result <- utils.SchemaData{Schema: schema, Tables: tableData}
	}
}

func fetchSchemas(db *sql.DB, sqlLogCh chan<- string) ([]string, error) {
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

func fetchTables(db *sql.DB, schema string, sqlLogCh chan<- string) ([]string, error) {
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

func fetchColumns(db *sql.DB, schema, tableName string, sqlLogCh chan<- string) ([]string, error) {
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

type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue *string
}

type FKInfo struct {
	Column     string
	RefSchema  string
	RefTable   string
	RefColumn  string
	Constraint string
}

type IndexInfo struct {
	Name       string
	Definition string
	IsUnique   bool
}

func fetchColumnDetails(db *sql.DB, schema, table string, sqlLogCh chan<- string) ([]ColumnInfo, error) {
	q := `
SELECT
  c.column_name,
  c.data_type,
  (c.is_nullable = 'YES') AS is_nullable,
  c.column_default
FROM information_schema.columns c
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position`
	sqlLogCh <- q
	rows, err := db.Query(q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ColumnInfo
	for rows.Next() {
		var (
			name, dataType string
			isNullable     bool
			def            sql.NullString
		)
		if err := rows.Scan(&name, &dataType, &isNullable, &def); err != nil {
			return nil, err
		}
		var defPtr *string
		if def.Valid {
			v := def.String
			defPtr = &v
		}
		out = append(out, ColumnInfo{
			Name:         name,
			DataType:     dataType,
			IsNullable:   isNullable,
			DefaultValue: defPtr,
		})
	}
	return out, rows.Err()
}

func fetchPrimaryKeys(db *sql.DB, schema, table string, sqlLogCh chan<- string) (map[string]bool, error) {
	q := `
SELECT a.attname AS column_name
FROM   pg_index i
JOIN   pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
JOIN   pg_class t ON t.oid = i.indrelid
JOIN   pg_namespace n ON n.oid = t.relnamespace
WHERE  i.indisprimary
AND    n.nspname = $1
AND    t.relname = $2`
	sqlLogCh <- q
	rows, err := db.Query(q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		out[col] = true
	}
	return out, rows.Err()
}

func fetchForeignKeys(db *sql.DB, schema, table string, sqlLogCh chan<- string) (map[string]FKInfo, error) {
	q := `
SELECT
  kcu.column_name,
  ccu.table_schema  AS ref_schema,
  ccu.table_name    AS ref_table,
  ccu.column_name   AS ref_column,
  tc.constraint_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
  AND tc.table_schema   = kcu.table_schema
JOIN information_schema.constraint_column_usage ccu
  ON ccu.constraint_name = tc.constraint_name
  AND ccu.table_schema   = tc.table_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = $1
  AND tc.table_name   = $2
ORDER BY kcu.position_in_unique_constraint`
	sqlLogCh <- q
	rows, err := db.Query(q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]FKInfo{}
	for rows.Next() {
		var info FKInfo
		if err := rows.Scan(&info.Column, &info.RefSchema, &info.RefTable, &info.RefColumn, &info.Constraint); err != nil {
			return nil, err
		}
		out[info.Column] = info
	}
	return out, rows.Err()
}

func fetchIndexes(db *sql.DB, schema, table string, sqlLogCh chan<- string) ([]IndexInfo, error) {
	q := `
SELECT
  i.indexname,
  i.indexdef,
  idx.indisunique AS is_unique
FROM pg_indexes i
JOIN pg_class t       ON t.relname = i.tablename
JOIN pg_namespace n   ON n.oid = t.relnamespace AND n.nspname = i.schemaname
JOIN pg_class ic      ON ic.relname = i.indexname
JOIN pg_index idx     ON idx.indexrelid = ic.oid
WHERE i.schemaname = $1
  AND i.tablename  = $2
ORDER BY i.indexname`
	sqlLogCh <- q
	rows, err := db.Query(q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []IndexInfo
	for rows.Next() {
		var (
			name, def string
			unique    bool
		)
		if err := rows.Scan(&name, &def, &unique); err != nil {
			return nil, err
		}
		out = append(out, IndexInfo{
			Name:       name,
			Definition: def,
			IsUnique:   unique,
		})
	}
	return out, rows.Err()
}

func fetchTableMetadata(db *sql.DB, schema, table string, sqlLogCh chan<- string) (cols []ColumnInfo, pks map[string]bool, fks map[string]FKInfo, idx []IndexInfo, err error) {
	cols, err = fetchColumnDetails(db, schema, table, sqlLogCh)
	if err != nil {
		return
	}
	pks, err = fetchPrimaryKeys(db, schema, table, sqlLogCh)
	if err != nil {
		return
	}
	fks, err = fetchForeignKeys(db, schema, table, sqlLogCh)
	if err != nil {
		return
	}
	idx, err = fetchIndexes(db, schema, table, sqlLogCh)
	return
}

func storeData(db *gorm.DB, dbID, userID string, ch <-chan utils.SchemaData, errCh chan<- error) {
	for data := range ch {
		var schemaID string
		if err := db.Raw(
			`SELECT id FROM schemas WHERE name = ? AND database_id = ? AND deleted_at IS NULL LIMIT 1`,
			data.Schema, dbID,
		).Scan(&schemaID).Error; err != nil {
			errCh <- err
			continue
		}

		if schemaID != "" {
			if err := db.Exec(`UPDATE schemas SET name = ?, updated_at = NOW() WHERE id = ?`, data.Schema, schemaID).Error; err != nil {
				errCh <- err
				continue
			}
		} else {
			newID := strings.ReplaceAll(uuid.New().String(), "-", "")
			if err := db.Raw(
				`INSERT INTO schemas (id, database_id, name, user_id, created_at, updated_at)
				 VALUES (?, ?, ?, ?, NOW(), NOW()) RETURNING id`,
				newID, dbID, data.Schema, userID,
			).Scan(&schemaID).Error; err != nil {
				errCh <- err
				continue
			}
		}

		for _, tbl := range data.Tables {
			var tableID string
			if err := db.Raw(
				`SELECT id FROM tables WHERE name = ? AND schema_id = ? AND deleted_at IS NULL LIMIT 1`,
				tbl.Table, schemaID,
			).Scan(&tableID).Error; err != nil {
				errCh <- err
				continue
			}

			if tableID != "" {
				if err := db.Exec(`UPDATE tables SET name = ?, updated_at = NOW() WHERE id = ?`, tbl.Table, tableID).Error; err != nil {
					errCh <- err
					continue
				}
			} else {
				newID := strings.ReplaceAll(uuid.New().String(), "-", "")
				if err := db.Raw(
					`INSERT INTO tables (id, schema_id, name, database_id, user_id, created_at, updated_at)
					 VALUES (?, ?, ?, ?, ?, NOW(), NOW()) RETURNING id`,
					newID, schemaID, tbl.Table, dbID, userID,
				).Scan(&tableID).Error; err != nil {
					errCh <- err
					continue
				}
			}

			for _, col := range tbl.Columns {
				var colID string
				if err := db.Raw(
					`SELECT id FROM columns WHERE name = ? AND table_id = ? AND deleted_at IS NULL LIMIT 1`,
					col.Name, tableID,
				).Scan(&colID).Error; err != nil {
					errCh <- err
					continue
				}

				if colID != "" {
					if err := db.Exec(
						`UPDATE columns
						   SET name = ?, data_type = ?, is_nullable = ?, default_value = ?, is_primary_key = ?,
						       fk_ref_schema = ?, fk_ref_table = ?, fk_ref_column = ?, fk_constraint = ?, updated_at = NOW()
						 WHERE id = ?`,
						col.Name, col.DataType, col.IsNullable, col.DefaultValue, col.IsPrimaryKey,
						col.FKRefSchema, col.FKRefTable, col.FKRefColumn, col.FKConstraint, colID,
					).Error; err != nil {
						errCh <- err
						continue
					}
				} else {
					newID := strings.ReplaceAll(uuid.New().String(), "-", "")
					if err := db.Exec(
						`INSERT INTO columns
						     (id, table_id, name, data_type, is_nullable, default_value, is_primary_key,
						      fk_ref_schema, fk_ref_table, fk_ref_column, fk_constraint, user_id, created_at, updated_at)
						 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
						newID, tableID, col.Name, col.DataType, col.IsNullable, col.DefaultValue, col.IsPrimaryKey,
						col.FKRefSchema, col.FKRefTable, col.FKRefColumn, col.FKConstraint, userID,
					).Error; err != nil {
						errCh <- err
						continue
					}
				}
			}

			for _, ix := range tbl.Indexes {
				var idxID string
				if err := db.Raw(
					`SELECT id FROM indexes WHERE name = ? AND table_id = ? AND deleted_at IS NULL LIMIT 1`,
					ix.Name, tableID,
				).Scan(&idxID).Error; err != nil {
					errCh <- err
					continue
				}

				if idxID != "" {
					if err := db.Exec(
						`UPDATE indexes
						   SET name = ?, definition = ?, is_unique = ?, updated_at = NOW()
						 WHERE id = ?`,
						ix.Name, ix.Definition, ix.IsUnique, idxID,
					).Error; err != nil {
						errCh <- err
						continue
					}
				} else {
					newID := strings.ReplaceAll(uuid.New().String(), "-", "")
					if err := db.Exec(
						`INSERT INTO indexes
						     (id, table_id, user_id, name, definition, is_unique, created_at, updated_at)
						 VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
						newID, tableID, userID, ix.Name, ix.Definition, ix.IsUnique,
					).Error; err != nil {
						errCh <- err
						continue
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
