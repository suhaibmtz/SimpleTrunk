// SimpleTrunk project main.go
package main

import (
	"html/template"
	"net/http"
	"strings"
)

var mytemplate *template.Template
var PREFIX string

const Version = "1.0.1 20Jan"

func main() {

	PREFIX = GetConfigValue("prefix", "/SimpleTrunk")
	if !strings.HasPrefix(PREFIX, "/") {
		PREFIX = "/" + PREFIX
	}
	prefix := PREFIX
	mytemplate = template.Must(template.ParseGlob("*templates/*.html"))
	http.Handle(prefix+"/static/", http.StripPrefix(prefix+"/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc(prefix+"/", Index)
	http.HandleFunc(prefix, redirect)
	http.HandleFunc("/", redirect)

	//Login
	http.HandleFunc(prefix+"/login", Login)
	http.HandleFunc(prefix+"/Login", Login)
	http.HandleFunc(prefix+"/Logout", Logout)

	//Home
	http.HandleFunc(prefix+"/Home", Home)
	http.HandleFunc(prefix+"/AddPBX", AddPBX)
	http.HandleFunc(prefix+"/EditPBX", EditPBX)
	http.HandleFunc(prefix+"/SelectPBX", SelectPBX)

	//Advanced
	http.HandleFunc(prefix+"/Advanced", Advanced)
	http.HandleFunc(prefix+"/Status", Status)
	http.HandleFunc(prefix+"/SIPNodes", SIPNodes)
	http.HandleFunc(prefix+"/EditNode", EditNode)
	http.HandleFunc(prefix+"/Dialplan", Dialplan)
	http.HandleFunc(prefix+"/Commands", Commands)
	http.HandleFunc(prefix+"/AMI", AMI)
	http.HandleFunc(prefix+"/Terminal", Terminal)
	http.HandleFunc(prefix+"/Logs", Logs)
	http.HandleFunc(prefix+"/Config", Config)
	http.HandleFunc(prefix+"/Backup", Backup)
	http.HandleFunc(prefix+"/AMIConfig", AMIConfig)
	http.HandleFunc(prefix+"/CDRConfig", CDRConfig)
	//Advanced Files
	http.HandleFunc(prefix+"/Files", Files)
	http.HandleFunc(prefix+"/BackupFiles", BackupFiles)
	http.HandleFunc(prefix+"/CompareFiles", CompareFiles)
	http.HandleFunc(prefix+"/EditFile", EditFile)
	//Sound
	http.HandleFunc(prefix+"/UploadSound", UploadSound)
	http.HandleFunc(prefix+"/PlaySound", PlaySound)
	http.HandleFunc(prefix+"/UploadSoundFile", UploadSoundFile)

	//PBX
	http.HandleFunc(prefix+"/PBX", PBX)
	http.HandleFunc(prefix+"/Extensions", Extensions)
	http.HandleFunc(prefix+"/Dialplans", Dialplans)
	http.HandleFunc(prefix+"/Functions", Functions)
	http.HandleFunc(prefix+"/Monitor", Monitor)

	//Admin
	http.HandleFunc(prefix+"/Admin", Admin)

	println("http://localhost:10025" + prefix)
	err := http.ListenAndServe(":10025", nil)
	if err != nil {
		println(err.Error())
	}
}

func redirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, PREFIX+"/", http.StatusTemporaryRedirect)
}
