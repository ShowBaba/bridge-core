package db

import (
	"context"
	"database/sql"
	"log"
	"sync"

	"github.com/showbaba/query-bridge/bridge/utils"
	"github.com/showbaba/query-bridge/shared"
)

func Migrate(db *sql.DB) {
	const TOTAL_WORKERS = 1
	var (
		wg      sync.WaitGroup
		errorCh = make(chan error, TOTAL_WORKERS)
	)
	wg.Add(TOTAL_WORKERS)
	log.Println("running db migration")

	go func() {
		defer wg.Done()
		// Users
		ctx, cancel := context.WithTimeout(context.Background(), utils.DbTimeout)
		defer cancel()
		tableExist, err := shared.CheckTableExist(ctx, db, "users")
		if err != nil {
			errorCh <- err
		}
		if !tableExist {
			query := `CREATE TABLE USERS(
					ID SERIAL PRIMARY KEY, EMAIL VARCHAR(255) UNIQUE NOT NULL,
					FIRSTNAME VARCHAR(255) NOT NULL, LASTNAME VARCHAR(255) NOT NULL,
					PASSWORD VARCHAR(255) NOT NULL, CREATED_AT DATE, UPDATED_AT DATE
				)`
			_, err := db.ExecContext(ctx, query)
			if err != nil {
				errorCh <- err
			}
		}
	}()

	go func() {
		wg.Wait()
		close(errorCh)
	}()

	for err := range errorCh {
		if err != nil {
			panic(err)
		}
	}

	log.Println("complete db migration")
}
