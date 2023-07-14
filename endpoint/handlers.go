package endpoint

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
)

func CreateEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	userId := r.Context().Value("id").(uint)

	var input CreateEndpointInput
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

	err = input.ValidateMethod()
	if err != nil {
		utils.Dispatch400Error(w, fmt.Sprintf("method validation error: %v", err))
		return
	}

	if input.OrderDirection != "" {
		err = input.ValidateOrderDirection()
		if err != nil {
			utils.Dispatch400Error(w, fmt.Sprintf("order_direction validation error: %v", err))
			return
		}
	}

	var application *models.Application
	application, exist, err := application.FetchApplication(db, models.Application{ID: input.ApplicationID})
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

	var table *models.Table
	table, exist, err = table.FetchTable(db, models.Table{ID: input.TableID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find table")
		return
	}

	var column models.Column
	columns, err := column.FetchColumns(db, models.Column{TableID: table.ID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if len(columns) == 0 {
		utils.Dispatch400Error(w, "table has no columns")
		return
	}
	if input.OrderBy != "" {
		// validate the column exist
		err := ValidateOrderByColumnExist(columns, input.OrderBy)
		if err != nil {
			utils.Dispatch400Error(w, err.Error())
			return
		}
	}

	var endpoint *models.Endpoint
	_, exist, err = endpoint.FetchEndpoint(db, models.Endpoint{
		ApplicationID: application.ID,
		TableID:       table.ID,
		Name:          input.Name,
	})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if exist {
		utils.Dispatch400Error(w, "endpoint with name already exist in this application and table")
		return
	}

	identifierUUID := strings.ReplaceAll(uuid.New().String(), "-", "")
	url := fmt.Sprintf(`%s/endpoint/execute/%s`, utils.GetConfig().ServerBaseURL, identifierUUID) // public access url
	var query string
	switch input.Method {
	case "GET":
		if len(input.Columns) == 0 {
			query = fmt.Sprintf(`SELECT * FROM %s`, table.Name)
		} else {
			query = fmt.Sprintf("SELECT %s FROM %s", strings.Join(input.Columns, ", "), table.Name)
		}

		if input.OrderBy != "" {
			query += fmt.Sprintf(` ORDER BY %s`, input.OrderBy)
		}

		if input.OrderDirection != "" {
			query += fmt.Sprintf(` %s`, input.OrderDirection)
		}

		if input.Limit != 0 {
			query += fmt.Sprintf(" LIMIT %v", input.Limit)
		}

	case "POST":
		valuePlaceholders := make([]string, len(input.Columns))
		for i := range input.Columns {
			valuePlaceholders[i] = fmt.Sprintf(`$%v`, i+1)
		}
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.Name, strings.Join(input.Columns, ", "), strings.Join(valuePlaceholders, ", "))
	}

	endpoint = &models.Endpoint{
		Name:           input.Name,
		ApplicationID:  input.ApplicationID,
		TableID:        input.TableID,
		Query:          query,
		UserID:         userId,
		IdentifierUUID: identifierUUID,
		Method:         input.Method,
		Limit:          input.Limit,
		IsPublic:       utils.BoolPointer(*input.IsPublic),
		OrderBy:        input.OrderBy,
		OrderDirection: input.OrderDirection,
		Columns:        input.Columns,
	}
	err = endpoint.Insert(db)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}

	response := utils.APIResponse{
		Status:  http.StatusCreated,
		Message: "endpoint created successfully",
		Data:    map[string]string{"url": url},
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}

// this endpoint will be used to execute a stored procedure
func ExecuteEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var input ExecuteEndpointInput
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
	identifier, ok := vars["identifier"]
	if !ok || identifier == "" {
		utils.Dispatch400Error(w, "missing identifier in request")
		return
	}
	userId := r.Context().Value("id").(uint)

	var endpoint *models.Endpoint
	endpoint, exist, err := endpoint.FetchEndpoint(db, models.Endpoint{
		IdentifierUUID: identifier,
		UserID:         userId,
	})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find endpoint with identifier")
		return
	}

	var application *models.Application
	application, exist, err = application.FetchApplication(db, models.Application{ID: endpoint.ApplicationID, UserID: userId})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find application")
		return
	}

	// validate if endpoint is not public ie it requires apiKey
	if !*endpoint.IsPublic {
		rawApiKey, err := utils.Decrypt(application.ApiKey, []byte(utils.GetConfig().EncryptionKey))
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		valid, err := ValidateApiKeyInReq(r, string(rawApiKey))
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		if !valid {
			utils.Dispatch401Error(w, "invalid/missing api_key in request header")
			return
		}
	}

	var table *models.Table
	table, exist, err = table.FetchTable(db, models.Table{ID: endpoint.TableID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find table")
		return
	}

	var schema *models.Schema
	schema, exist, err = schema.FetchSchema(db, models.Schema{ID: table.SchemaID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find schema associated with table")
		return
	}

	var databaseInfo *models.Database
	databaseInfo, exist, err = databaseInfo.FetchDatabase(db, models.Database{ID: schema.DatabaseID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find database associated with schema")
		return
	}

	dbConn, err := OpenSqlxConnection(databaseInfo)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}

	switch endpoint.Method {
	case "POST":
		// validate the length of values
		if len(input.Values) != len(endpoint.Columns) {
			utils.Dispatch400Error(w, "invalid number of values in request")
			return
		}
		err := ExecuteInsertQuery(dbConn, endpoint.Query, input.Values)
		if err != nil {
			utils.Dispatch500Error(w, fmt.Sprintf("failed to execute query: %v", err))
			return
		}
		var stream *models.StreamLog
		err = stream.Insert(ctx, mongoClient, endpoint.Query, endpoint.ApplicationID, endpoint.UserID)
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		response := utils.APIResponse{
			Status:  http.StatusCreated,
			Message: "query executed successfully",
		}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		w.Write(responseJSON)
		return
	case "GET":
		fetchResult, err := ExecuteFetchQuery(dbConn, endpoint.Query, endpoint.Columns)
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		var stream *models.StreamLog
		err = stream.Insert(ctx, mongoClient, endpoint.Query, endpoint.ApplicationID, endpoint.UserID)
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}

		response := utils.APIResponse{
			Status:  http.StatusOK,
			Message: "query executed successfully",
			Data:    fetchResult,
		}
		responseJSON, err := json.Marshal(response)
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		w.Write(responseJSON)
		return
	}
}

func UpdateEndpointHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var input UpdateEndpointInput
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

	if input.OrderDirection != "" {
		err = input.ValidateOrderDirection()
		if err != nil {
			utils.Dispatch400Error(w, fmt.Sprintf("order_direction validation error: %v", err))
			return
		}
	}

	vars := mux.Vars(r)
	endpoint_id, ok := vars["endpoint_id"]
	if !ok || endpoint_id == "" {
		utils.Dispatch400Error(w, "missing endpoint_id in request")
		return
	}

	endpoint_id_num, err := strconv.ParseUint(endpoint_id, 10, 64)
	if err != nil {
		fmt.Println("Failed to convert string to uint:", err)
		return
	}

	userId := r.Context().Value("id").(uint)
	var endpoint *models.Endpoint
	endpoint, exist, err := endpoint.FetchEndpoint(db, models.Endpoint{
		ID:     uint(endpoint_id_num),
		UserID: userId,
	})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !exist {
		utils.Dispatch404Error(w, "cannot find endpoint with identifier")
		return
	}

	if input.Name != "" {
		_, exist, err = endpoint.FetchEndpoint(db, models.Endpoint{
			ApplicationID: endpoint.ApplicationID,
			TableID:       endpoint.TableID,
			Name:          input.Name,
		})
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		if exist {
			utils.Dispatch400Error(w, "endpoint with name already exist in this application and table")
			return
		}
	}

	var table *models.Table
	if input.TableID != 0 {
		table, exist, err = table.FetchTable(db, models.Table{ID: input.TableID})
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		if !exist {
			utils.Dispatch404Error(w, "cannot find table")
			return
		}
	} else {
		table, exist, err = table.FetchTable(db, models.Table{ID: endpoint.TableID})
		if err != nil {
			utils.Dispatch500Error(w, err.Error())
			return
		}
		if !exist {
			utils.Dispatch404Error(w, "cannot find table")
			return
		}
	}

	var column models.Column
	columns, err := column.FetchColumns(db, models.Column{TableID: table.ID})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if len(columns) == 0 {
		utils.Dispatch400Error(w, "table has no columns")
		return
	}
	if input.OrderBy != "" {
		// validate the column exist
		err := ValidateOrderByColumnExist(columns, input.OrderBy)
		if err != nil {
			utils.Dispatch400Error(w, err.Error())
			return
		}
	}
	var query string
	if input.Method != "" {
		switch input.Method {
		case "GET":
			if len(input.Columns) == 0 {
				query = fmt.Sprintf(`SELECT * FROM %s`, table.Name)
			} else {
				query = fmt.Sprintf("SELECT %s FROM %s", strings.Join(input.Columns, ", "), table.Name)
			}
		case "POST":
			valuePlaceholders := make([]string, len(input.Columns))
			for i := range input.Columns {
				valuePlaceholders[i] = fmt.Sprintf(`$%v`, i+1)
			}
			query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table.Name, strings.Join(input.Columns, ", "), strings.Join(valuePlaceholders, ", "))
		}
	}

	if input.Method == "" {
		input.Method = endpoint.Method
	}
	if input.TableID == 0 {
		input.TableID = endpoint.TableID
	}
	if input.Name == "" {
		input.Name = endpoint.Name
	}
	if len(input.Columns) == 0 {
		input.Columns = endpoint.Columns
	}

	update := models.Endpoint{
		Name:    input.Name,
		TableID: input.TableID,
		Method:  input.Method,
		Columns: input.Columns,
		Query:   query,
	}
	if input.IsPublic != nil {
		if *input.IsPublic != *endpoint.IsPublic {
			update.IsPublic = utils.BoolPointer(*input.IsPublic)
		}
	}
	err = endpoint.Update(db, update)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	url := fmt.Sprintf(`%s/endpoint/execute/%s`, utils.GetConfig().ServerBaseURL, endpoint.IdentifierUUID)
	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "endpoint updated successfully",
		Data:    map[string]string{"url": url},
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
