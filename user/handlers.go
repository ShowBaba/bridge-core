package user

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-playground/validator"
	// "github.com/showbaba/query-bridge/bridge/data"
	"github.com/showbaba/query-bridge/bridge/models"
	"github.com/showbaba/query-bridge/shared"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	var input RegisterPayload
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
		shared.Dispatch400Error(w, "validation error", validationErrors.Error())
		return
	}
	var user models.User
	userData, err := user.GetByEmail(db, input.Email)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	if userData != nil {
		shared.Dispatch400Error(w, "email already used", nil)
		return
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	user = models.User{
		Email:     input.Email,
		Firstname: input.Firstname,
		Lastname:  input.Lastname,
		Password:  hash,
	}
	_, err = user.Insert(db)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	mail := shared.Mail{
		Sender:  shared.MAIL_USERNAME,
		Subject: "Welcome to our blog!",
		To:      []string{input.Email},
		Body: `<div style="font-family: Helvetica, Arial, sans-serif; min-width: 1000px; overflow: auto; line-height: 2;">
            <div style="margin: 50px auto; width: 70%; padding: 20px 0;">
                <div style="border-bottom: 1px solid #eee;"><a href="blog.com" style="font-size: 1.4em; color: #00466a; text-decoration: none; font-weight: 600;">SAM's BLOG</a></div>
                <p style="font-size: 1.1em;">Hi,</p>
                <p>Hi ` + input.Firstname + `</p>
                <p>Welcome to Sam's BLOG</p>
                <p style="font-size: 0.9em;">
                    Regards,<br />
                    SAM's BLOG
                </p>
                <hr style="border: none; border-top: 1px solid #eee;" />
            </div>
        </div>`,
	}
	payload, err := json.Marshal(mail)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	if err := shared.SendNotification(messageChan, payload); err != nil {
		if err != nil {
			shared.Dispatch500Error(w, err)
			return
		}
	}
	response := shared.APIResponse{
		Status:  http.StatusOK,
		Message: "user registered successfully",
		Data:    nil,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		shared.Dispatch500Error(w, err)
		return
	}
	w.Write(responseJSON)
}
