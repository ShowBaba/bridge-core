package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/showbaba/query-bridge/bridge-core/models"
	"github.com/showbaba/query-bridge/bridge-core/utils"
)

// re-fetch database information
func UpdateDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	userId := r.Context().Value("id").(uint)
	var input UpdateDatabasePayload
	if body, err := io.ReadAll(r.Body); err != nil {
		utils.Dispatch400Error(w, "invalid body: %s")
		return
	} else if err := json.Unmarshal(body, &input); err != nil {
		utils.Dispatch400Error(w, "invalid body: %s")
		return
	}
	validate := validator.New()
	err := validate.Struct(input)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		utils.Dispatch400Error(w, validationErrors.Error())
		return
	}
	vars := mux.Vars(r)
	databaseIDStr, ok := vars["database_id"]

	if !ok || databaseIDStr == "" {
		utils.Dispatch400Error(w, "invalid or missing database ID")
		return
	}

	databaseID, err := strconv.Atoi(databaseIDStr)
	if err != nil {
		utils.Dispatch400Error(w, "invalid database ID format")
		return
	}

	var database *models.Database

	database, exist, err := database.FetchDatabase(db, models.Database{ID: uint(databaseID)})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "database with id not found")
		return
	}
	var application *models.Application
	_, exist, err = application.FetchApplication(db, models.Application{ID: database.ApplicationID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find application")
		return
	}
	if application.UserID != userId {
		utils.Dispatch401Error(w, "unauthorized")
		return
	}

	if input.Database != "" {
		if input.Host == "" || input.Host != database.Host {
			_, exist, err := database.FetchDatabase(db, models.Database{Database: input.Database, Host: database.Host, ApplicationID: database.ApplicationID})
			if err != nil {
				utils.Dispatch500Error(w, err.Error())
				return
			}
			if exist {
				utils.Dispatch400Error(w, "duplicate database name")
				return
			}
		}
	}
	// test the connection in case any crucial changes was made
	if input.Host == "" {
		input.Host = database.Host
	}
	if input.Port == 0 {
		input.Port = database.Port
	}
	if input.Database == "" {
		input.Database = database.Database
	}
	if input.Username == "" {
		input.Username = database.Username
	}
	if input.Password == "" {
		rawPassword, err := utils.Decrypt(database.Password, []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		input.Password = string(rawPassword)
	}
	if input.DbEngine == "" {
		input.DbEngine = database.DbEngine
	}

	dbConn, err := utils.TestDatabaseConnection(utils.DatabaseConnectionPayload{
		Host:     input.Host,
		Port:     input.Port,
		Database: input.Database,
		Username: input.Username,
		Password: input.Password,
		DbEngine: input.DbEngine,
	})
	if err != nil {
		utils.Dispatch400Error(w, fmt.Sprintf("error creating database connection: %v", err))
		return
	}
	// close the database connection
	dbConn.Close()

	// if password is in update encrypt
	encryptedPassword, err := utils.Encrypt([]byte(input.Password), []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	err = database.Update(db, map[string]interface{}{
		"Host":     input.Host,
		"Port":     input.Port,
		"Database": input.Database,
		"Username": input.Username,
		"Password": encryptedPassword,
		"DbEngine": input.DbEngine,
	})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}

	//TODO: only need to re-fetch database data if only credentials changed
	databaseTask := utils.DatabaseTask{
		DatabaseID: uint(databaseID),
		UserID:     userId,
	}
	payload, err := json.Marshal(databaseTask)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if err := utils.PublishMessageToQueue(ctx, queueConnection, payload, utils.DATABASE_QUEUE); err != nil {
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
	}
	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "database updated successfully",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}

func DeleteDatabases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	userId := r.Context().Value("id").(uint)

	vars := mux.Vars(r)
	databaseIDStr, ok := vars["database_id"]

	if !ok || databaseIDStr == "" {
		utils.Dispatch400Error(w, "invalid or missing database ID")
		return
	}

	databaseID, err := strconv.Atoi(databaseIDStr)
	if err != nil {
		utils.Dispatch400Error(w, "invalid database ID format")
		return
	}

	var database *models.Database

	database, exist, err := database.FetchDatabase(db, models.Database{ID: uint(databaseID)})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "database with id not found")
		return
	}

	var application *models.Application
	application, exist, err = application.FetchApplication(db, models.Application{ID: database.ApplicationID, UserID: userId})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}

	if !exist {
		utils.Dispatch404Error(w, "cannot find application")
		return
	}

	if application.UserID != userId {
		utils.Dispatch401Error(w, "unauthorized")
		return
	}

	// delete other associating resources
	go func() {
		var database models.Database
		databases, err := database.FetchDatabases(db, models.Database{ApplicationID: application.ID})
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}

		var (
			databaseIDs []uint
			schemaIDs   []uint
			tableIDs    []uint
			columnIDs   []uint
		)

		for _, database := range databases {
			databaseIDs = append(databaseIDs, database.ID)

			var schema models.Schema
			schemas, err := schema.FetchSchemas(db, models.Schema{DatabaseID: database.ID})
			if err != nil {
				utils.Dispatch500Error(w, err.Error())
				return
			}

			for _, schema := range schemas {
				schemaIDs = append(schemaIDs, schema.ID)

				var table models.Table
				tables, err := table.FetchTables(db, models.Table{SchemaID: schema.ID})
				if err != nil {
					utils.Dispatch500Error(w, err.Error())
					return
				}

				for _, table := range tables {
					tableIDs = append(tableIDs, table.ID)

					var column models.Column
					columns, err := column.FetchColumns(db, models.Column{TableID: table.ID})
					if err != nil {
						utils.Dispatch500Error(w, err.Error())
						return
					}

					for _, column := range columns {
						columnIDs = append(columnIDs, column.ID)
					}
				}
			}
		}

		var column *models.Column
		if err := column.DeleteMany(db, columnIDs); err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}

		var table *models.Table
		if err := table.DeleteMany(db, tableIDs); err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}

		var schema *models.Schema
		if err := schema.DeleteMany(db, schemaIDs); err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}

		if err := database.DeleteMany(db, databaseIDs); err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
	}()

	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "database deleted successfully",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
