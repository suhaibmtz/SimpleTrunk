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
	Head.Tabs = append(Head.Tabs, TabsType{Tabs: pbxPages, Selected: selected})
	return
}

func PBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Header := GetPBXHeader(User.Name, "PBX", "", r)
		err := mytemplate.ExecuteTemplate(w, "pbxpage.html", Header)
		if err != nil {
			WriteLog("Error in Home execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
