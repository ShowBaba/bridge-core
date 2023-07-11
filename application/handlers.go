package application

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

func CreateApplication(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input CreateApplicationPayload
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
	// validate duplicate application name for a user
	var application *models.Application
	userId := r.Context().Value("id").(uint)

	application, err = application.FetchApplication(db, models.Application{Name: input.Name, UserID: userId})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if application != nil {
		utils.Dispatch400Error(w, "duplicate application name")
		return
	}
	application = &models.Application{
		Name:   input.Name,
		UserID: userId,
	}
	err = application.Insert(db)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	response := utils.APIResponse{
		Status:  http.StatusCreated,
		Message: "application created successfully",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}

func AddDatabases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input AddDatabasePayload
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
	applicationIDStr, ok := vars["application_id"]

	// Validate application_id
	if !ok || applicationIDStr == "" {
		utils.Dispatch400Error(w, "invalid or missing application ID")
		return
	}

	applicationID, err := strconv.Atoi(applicationIDStr)
	if err != nil {
		utils.Dispatch400Error(w, "invalid application ID format")
		return
	}

	// validate duplicate db name
	var database *models.Database

	_, exist, err := database.FetchDatabase(db, models.Database{Name: input.Name, ApplicationID: uint(applicationID)})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if exist {
		utils.Dispatch400Error(w, "duplicate database name")
		return
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
		utils.Dispatch400Error(w, fmt.Sprintf(`error creating database connection; err: (%v)`, err))
		return
	}

	// close the database connection
	dbConn.Close()

	encryptedPassword, err := utils.Encrypt([]byte(input.Password), []byte(utils.GetConfig().EncryptionKey))
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	database = &models.Database{
		Name:          input.Name,
		Host:          input.Host,
		Port:          input.Port,
		Database:      input.Database,
		Username:      input.Username,
		Password:      encryptedPassword,
		DbEngine:      input.DbEngine,
		ApplicationID: uint(applicationID),
	}
	id, err := database.Insert(db)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	// fetch db schema and schema tables asynchronously
	databaseTask := utils.DatabaseTask{
		DatabaseID: id,
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
		Message: "database created successfully",
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)

}


