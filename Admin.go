package main

import (
	"net/http"
)

type AdminType struct {
	HeaderType
	OldPass  string
	NewPass  string
	ConfPass string
	Reverse  bool
}

func Admin(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		var Data AdminType
		Data.HeaderType = GetHeader(User, "My Admin", r)
		Data.OldPass = r.FormValue("oldpassword")
		Data.NewPass = r.FormValue("newpassword")
		Data.ConfPass = r.FormValue("confirmpassword")
		if r.FormValue("resetpassword") != "" {
			if Data.NewPass == "" {
				Data.Message = "Empty password"
				Data.MessageType = "errormessage"
			} else if Data.NewPass != Data.ConfPass {
				Data.Message = "Passwords do not match"
				Data.MessageType = "errormessage"
			} else if GetMD5(Data.OldPass) != User.Password {
				Data.Message = "Invalid password"
				Data.MessageType = "errormessage"
			} else {
				// change password
				success, message := UpdateUserPassword(User.ID, Data.NewPass)
				if success {
					Data.MessageType = "infomessage"
					Data.Message = "Password has been changed"
				} else {
					Data.Message = "Unable to change password: " + message
					Data.MessageType = "errormessage"
				}
			}
		}
		reverse := r.FormValue("reverse")
		if reverse != "" {
			co, err := r.Cookie("reverse")
			if err != nil {
				co = &http.Cookie{
					Name:   "reverse",
					MaxAge: 60 * 60 * 24 * 7,
					Value:  reverse,
				}
			} else {
				co.MaxAge = 60 * 60 * 24 * 7
				co.Value = reverse
			}
			http.SetCookie(w, co)
		} else {
			reverse = GetCookieValue(r, "reverse")
		}
		Data.Reverse = reverse == "yes"
		err := mytemplate.ExecuteTemplate(w, "admin.html", Data)
		if err != nil {
			WriteLog("Error in Admin execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
