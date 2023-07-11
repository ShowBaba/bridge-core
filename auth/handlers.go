package auth

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input LoginPayload
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
	var user *models.User
	user, err = user.GetUser(db, models.User{Email: input.Email})
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if user == nil {
		utils.Dispatch404Error(w, "email is not registered")
		return
	}
	passwordMatch, err := PasswordMatches(input.Password, user.Password)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if !passwordMatch {
		utils.Dispatch404Error(w, "incorrect password")
		return
	}
	jwtToken, err := GenerateToken(utils.GetConfig().JWTSecretKey, input.Email, user.ID)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	type Token struct {
		Token string `json:"token"`
	}
	token := Token{Token: jwtToken}
	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "login",
		Data:    token,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
