package auth

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
	"github.com/showbaba/query-bridge/shared"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input LoginPayload
	if body, err := io.ReadAll(r.Body); err != nil {
		shared.Dispatch400Error(w, "invalid body: %s", err)
		return
	} else if err := json.Unmarshal(body, &input); err != nil {
		shared.Dispatch400Error(w, "invalid body: %s", err)
		return
	}
	validate := validator.New()
	err := validate.Struct(input)
	if err != nil {
		validationErrors := err.(validator.ValidationErrors)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(shared.WriteError(http.StatusBadRequest, validationErrors.Error(), nil))
		return
	}
	var user models.User
	userData, err := user.GetByEmail(db, input.Email)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	if userData == nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write(shared.WriteError(http.StatusNotFound, "email is not registered", nil))
		return
	}
	passwordMatch, err := PasswordMatches(input.Password, userData.Password)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	if !passwordMatch {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(shared.WriteError(http.StatusBadRequest, "incorrect password", nil))
		return
	}
	jwtToken, err := GenerateToken(utils.GetConfig().JWTSecretKey, input.Email)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	type Token struct {
		Token string `json:"token"`
	}
	token := Token{Token: jwtToken}
	response := shared.APIResponse{
		Status:  http.StatusOK,
		Message: "login",
		Data:    token,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	w.Write(responseJSON)
}
