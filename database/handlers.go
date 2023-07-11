package database

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-playground/validator"
	"github.com/gorilla/mux"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
)

// re-fetch database information
func UpdateDatabase(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
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
	if input.Name != "" {
		_, exist, err := database.FetchDatabase(db, models.Database{Name: input.Name, ApplicationID: database.ApplicationID})
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		if exist {
			utils.Dispatch400Error(w, "duplicate database name")
			return
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
		Name:     input.Name,
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
		"Name":     input.Name,
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

	//TODO: re-fetch database data if only credentials changed
	databaseTask := utils.DatabaseTask{
		DatabaseID: uint(databaseID),
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
		Status:  http.StatusCreated,
		Message: "database updated successfully",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
