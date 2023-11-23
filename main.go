// SimpleTrunk project main.go
package main

import (
	"html/template"
	"net/http"
)

var mytemplate *template.Template

const Version = "0.6.25 16Nov"

func main() {
	mytemplate = template.Must(template.ParseGlob("templates/*.html"))
	http.Handle("/SimpleTrunk/static/", http.StripPrefix("/SimpleTrunk/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/SimpleTrunk/", Index)

	http.HandleFunc("/SimpleTrunk/login", Login)
	http.HandleFunc("/SimpleTrunk/Login", Login)
	http.HandleFunc("/SimpleTrunk/Logout", Logout)

	http.HandleFunc("/SimpleTrunk/Home", Home)
	http.HandleFunc("/SimpleTrunk/AddPBX", AddPBX)
	http.HandleFunc("/SimpleTrunk/EditPBX", EditPBX)
	http.HandleFunc("/SimpleTrunk/SelectPBX", SelectPBX)

	http.HandleFunc("/SimpleTrunk/Advanced", Advanced)
	http.HandleFunc("/SimpleTrunk/Status", Status)
	//files
	http.HandleFunc("/SimpleTrunk/Files", Files)
	http.HandleFunc("/SimpleTrunk/BackupFiles", BackupFiles)
	http.HandleFunc("/SimpleTrunk/CompareFiles", CompareFiles)
	http.HandleFunc("/SimpleTrunk/EditFile", EditFile)

	http.HandleFunc("/SimpleTrunk/PBX", PBX)

	http.HandleFunc("/SimpleTrunk/Admin", Admin)

	println("http://localhost:10025/SimpleTrunk")
	http.ListenAndServe(":10025", nil)
}
