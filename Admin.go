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
	Users    []UserType
}

func Admin(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		var Data AdminType
		Data.HeaderType = GetHeader(User, "My Admin", r)
		Data.OldPass = r.FormValue("oldpassword")
		Data.NewPass = r.FormValue("newpassword")
		Data.ConfPass = r.FormValue("confirmpassword")
		if r.FormValue("add") != "" {
			if User.Admin {
				Username := r.FormValue("user")
				Password := r.FormValue("password")
				IsAdmin := r.FormValue("admin") != ""
				_, success, message := AddUser(Username, Password, IsAdmin)
				if success == false {
					Data.ErrorMessage(message)
				} else {
					WriteLog(User.Name + " added " + Username)
					Data.InfoMessage("User " + Username + " added")
				}
			} else {
				Data.ErrorMessage("Not Admin")
			}
		}
		if User.Admin {
			users := CallGetUsers()
			sorted := false
			for !sorted {
				sorted = true
				for i := 0; i < len(users)-1; i++ {
					a := users[i]
					b := users[i+1]
					if !a.Admin && b.Admin {
						sorted = false
						users[i], users[i+1] = b, a
					}
				}
			}
			Data.Users = users
		}
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
