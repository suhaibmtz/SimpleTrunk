// SimpleTrunk project main.go
package main

import (
	"html/template"
	"net/http"
)

var mytemplate *template.Template

const Version = "1.0.0 25Dec"

func main() {
	mytemplate = template.Must(template.ParseGlob("*templates/*.html"))
	http.Handle("/SimpleTrunk/static/", http.StripPrefix("/SimpleTrunk/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/SimpleTrunk/", Index)

	//Login
	http.HandleFunc("/SimpleTrunk/login", Login)
	http.HandleFunc("/SimpleTrunk/Login", Login)
	http.HandleFunc("/SimpleTrunk/Logout", Logout)

	//Home
	http.HandleFunc("/SimpleTrunk/Home", Home)
	http.HandleFunc("/SimpleTrunk/AddPBX", AddPBX)
	http.HandleFunc("/SimpleTrunk/EditPBX", EditPBX)
	http.HandleFunc("/SimpleTrunk/SelectPBX", SelectPBX)

	//Advanced
	http.HandleFunc("/SimpleTrunk/Advanced", Advanced)
	http.HandleFunc("/SimpleTrunk/Status", Status)
	http.HandleFunc("/SimpleTrunk/SIPNodes", SIPNodes)
	http.HandleFunc("/SimpleTrunk/EditNode", EditNode)
	http.HandleFunc("/SimpleTrunk/Dialplan", Dialplan)
	http.HandleFunc("/SimpleTrunk/Commands", Commands)
	http.HandleFunc("/SimpleTrunk/AMI", AMI)
	http.HandleFunc("/SimpleTrunk/Terminal", Terminal)
	http.HandleFunc("/SimpleTrunk/Logs", Logs)
	http.HandleFunc("/SimpleTrunk/Config", Config)
	http.HandleFunc("/SimpleTrunk/Backup", Backup)
	http.HandleFunc("/SimpleTrunk/AMIConfig", AMIConfig)
	http.HandleFunc("/SimpleTrunk/CDRConfig", CDRConfig)
	//Advanced Files
	http.HandleFunc("/SimpleTrunk/Files", Files)
	http.HandleFunc("/SimpleTrunk/BackupFiles", BackupFiles)
	http.HandleFunc("/SimpleTrunk/CompareFiles", CompareFiles)
	http.HandleFunc("/SimpleTrunk/EditFile", EditFile)
	//Sound
	http.HandleFunc("/SimpleTrunk/UploadSound", UploadSound)
	http.HandleFunc("/SimpleTrunk/PlaySound", PlaySound)
	http.HandleFunc("/SimpleTrunk/UploadSoundFile", UploadSoundFile)

	//PBX
	http.HandleFunc("/SimpleTrunk/PBX", PBX)
	http.HandleFunc("/SimpleTrunk/Extensions", Extensions)
	http.HandleFunc("/SimpleTrunk/Dialplans", Dialplans)
	http.HandleFunc("/SimpleTrunk/Functions", Functions)
	http.HandleFunc("/SimpleTrunk/Monitor", Monitor)

	//Admin
	http.HandleFunc("/SimpleTrunk/Admin", Admin)

	println("http://localhost:10025/SimpleTrunk")
	err := http.ListenAndServe(":10025", nil)
	if err != nil {
		println(err.Error())
	}
}
