package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/bridge/utils"

	"gorm.io/gorm"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input RegisterPayload
	if body, err := io.ReadAll(r.Body); err != nil {
		utils.Dispatch400Error(w, fmt.Sprintf("invalid body: %s", err.Error()))
		return
	} else if err := json.Unmarshal(body, &input); err != nil {
		utils.Dispatch400Error(w, fmt.Sprintf("invalid body: %s", err.Error()))
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
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if user != nil {
		utils.Dispatch400Error(w, "email already used")
		return
	}
	hash, err := utils.HashPassword(input.Password)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	user = &models.User{
		Email:     input.Email,
		FirstName: input.Firstname,
		LastName:  input.Lastname,
		Password:  hash,
	}
	_, err = user.Insert(db)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	mail := utils.Mail{
		Sender:  utils.MAIL_USERNAME,
		Subject: "Welcome QueryBridge!",
		To:      []string{input.Email},
		Body: `<div style="font-family: Helvetica, Arial, sans-serif; min-width: 1000px; overflow: auto; line-height: 2;">
            <div style="margin: 50px auto; width: 70%; padding: 20px 0;">
                <div style="border-bottom: 1px solid #eee;"><a href="google.com" style="font-size: 1.4em; color: #00466a; text-decoration: none; font-weight: 600;">QueryBridge</a></div>
                <p style="font-size: 1.1em;">Hi,</p>
                <p>Hi ` + input.Firstname + `</p>
                <p>Welcome to QueryBridge</p>
                <p style="font-size: 0.9em;">
                    Regards,<br />
                    QueryBridge
                </p>
                <hr style="border: none; border-top: 1px solid #eee;" />
            </div>
        </div>`,
	}
	payload, err := json.Marshal(mail)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	if err := utils.PublishMessageToQueue(ctx, queueConnection, payload, utils.NOTIFICATION_QUEUE); err != nil {
		if err != nil {
			utils.Dispatch500Error(w,  err.Error())
			return
		}
	}
	response := utils.APIResponse{
		Status:  http.StatusOK,
		Message: "user registered successfully",
		Data:    nil,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		utils.Dispatch500Error(w, err.Error())
		return
	}
	w.Write(responseJSON)
}
