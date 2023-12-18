package main

import (
	"net/http"
)

var pbxPages = []TabType{
	{Name: "Extensions", Value: "Extensions"},
	{Value: "Extensions?type=trunk", Name: "Trunks"},
	{Value: "Dialplans", Name: "Dialplans"},
	{Value: "Functions", Name: "Queues"},
	{Value: "Monitor", Name: "Monitor"},
}

func GetPBXHeader(username, page, selected string, r *http.Request) (Head HeaderType) {
	Head = GetHeader(username, "PBX", r)
	Head.Tabs = append(Head.Tabs, TabsType{Tabs: pbxPages, Selected: page})
	return
}

func PBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Header := GetPBXHeader(User.Name, "PBX", "", r)
		err := mytemplate.ExecuteTemplate(w, "pbxpage.html", Header)
		if err != nil {
			WriteLog("Error in PBX execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type ExtensionsType struct {
	HeaderType
	IsExten bool
}

func Extensions(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		var Data ExtensionsType
		page := "Extensions"
		Type := r.FormValue("type")
		Data.IsExten = true
		if Type == "trunk" {
			page = "Trunks"
			Data.IsExten = false
		}
		Data.HeaderType = GetPBXHeader(User.Name, page, "", r)
		fileName := r.FormValue("file")
		if fileName == "" {
			fileName = "sip.conf"
		}

		err := mytemplate.ExecuteTemplate(w, "Extensions.html", Data)
		if err != nil {
			WriteLog("Error in Extensions execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
