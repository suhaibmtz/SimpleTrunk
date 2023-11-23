package main

import (
	"net/http"
)

func PBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Header := GetAdvancedHeader(User.Name, "PBX", "", r)
		err := mytemplate.ExecuteTemplate(w, "pbx.html", Header)
		if err != nil {
			WriteLog("Error in Home execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
