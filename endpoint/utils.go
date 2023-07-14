package endpoint

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
)

func OpenSqlxConnection(dbModel *models.Database) (*sqlx.DB, error) {
	rawPassword, err := utils.Decrypt(dbModel.Password, []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		return nil, err
	}
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%v password=%s user=%s dbname=%s sslmode=disable", dbModel.Host, dbModel.Port, rawPassword, dbModel.Username, dbModel.Database))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ExecuteInsertQuery(db *sqlx.DB, query string, values []interface{}) error {
	_, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}
	return nil
}

func ExecuteFetchQuery(db *sqlx.DB, query string, returnColumns []string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var scanColumns []interface{}
	if len(returnColumns) == 0 {
		scanColumns = make([]interface{}, len(columns))
		for i := range columns {
			scanColumns[i] = new(interface{})
		}
	} else {
		scanColumns = make([]interface{}, len(returnColumns))
		for i, col := range returnColumns {
			for _, c := range columns {
				if strings.EqualFold(c, col) {
					scanColumns[i] = new(interface{})
					break
				}
			}
		}
	}

	results := make([]map[string]interface{}, 0)

	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.Scan(scanColumns...)
		if err != nil {
			return nil, err
		}

		for i, col := range columns {
			if len(returnColumns) == 0 || contains(returnColumns, col) {
				row[col] = *(scanColumns[i].(*interface{}))
			}
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// contains checks if a string slice contains a specific value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func ValidateApiKeyInReq(r *http.Request, apiKey string) (bool, error) {
	reqHeaderKey := r.Header["X-Apikey"]
	if len(reqHeaderKey) == 0 || reqHeaderKey[0] != apiKey {
		return false, nil
	}
	return true, nil
}
