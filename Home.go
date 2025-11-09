package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type TabType struct {
	Name  string
	Value string
}

type PBXFileType struct {
	FileName string
	Path     string
	Index    int
	Title    string
	Color    string
	Url      string
	IP       string
	NewTR    bool
	IsStc    bool
}

type TabsType struct {
	Selected string
	Text     string
	Tabs     []TabType
}

type HeaderType struct {
	LogoutText  string
	SelectedPBX PBXFileType
	Version     string
	PBXFiles    []PBXFileType
	MainTabs    TabsType
	Tabs        []TabsType
	ShowPages   bool
	Message     string
	MessageType string
	IsAdmin     bool
}

func (h *HeaderType) ErrorMessage(message string) {
	h.Message = message
	h.MessageType = "errormessage"
}

func (h *HeaderType) InfoMessage(message string) {
	h.Message = message
	h.MessageType = "infomessage"
}

type LoginType struct {
	HeaderType
	Create     bool
	Login      string
	Password   string
	RememberMe bool
}

func Login(w http.ResponseWriter, r *http.Request) {
	var Data LoginType
	Data.HeaderType = GetHeader(UserType{}, "", r)
	Data.ShowPages = false
	Data.Create = len(CallGetUsers()) == 0
	if r.FormValue("log") != "" {
		Data.Login = r.FormValue("login")
		Data.Password = r.FormValue("pass")
		Data.RememberMe = r.FormValue("rememberme") != ""
		if Data.Create {
			_, success, message := AddUser(Data.Login, Data.Password, Data.Create)
			if !success {
				Data.Message = message
				Data.MessageType = "errormessage"
			} else {
				WriteLog("Success Login for: " + Data.Login)
				http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
			}
		} else {
			User, exist := GetUserByName(Data.Login)
			if exist && GetMD5(Data.Password) == User.Password {
				var key = GetMD5(time.Now().String())
				SetSession(key, User.ID)
				cookie := &http.Cookie{
					Name:  "st-session",
					Value: key,
				}
				if Data.RememberMe {
					cookie.MaxAge = 60 * 60 * 24 * 30
				}
				http.SetCookie(w, cookie)
				http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
			} else {
				Data.MessageType = "errormessage"
				Data.Message = "Invalid login/password"
			}
		}
	}
	err := mytemplate.ExecuteTemplate(w, "login.html", Data)
	if err != nil {
		WriteLog("Error in Login: " + err.Error())
	}
}

var taps = []TabType{{Name: "Home", Value: "Home"}, {Name: "Advanced", Value: "Advanced"}, {Name: "PBX", Value: "PBX"}, {Name: "My Admin", Value: "Admin"}}

func Index(w http.ResponseWriter, r *http.Request) {
	mytemplate.ExecuteTemplate(w, "index.html", nil)
}

type HomeType struct {
	HeaderType
	Files []PBXFileType
}

func Home(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		var Data HomeType
		Data.HeaderType = GetHeader(User, "Home", r)
		Data.Files = GetPBXFiles()
		message := r.FormValue("m")
		if message != "" {
			Data.Message = message
			Data.MessageType = "errormessage"
		}
		err := mytemplate.ExecuteTemplate(w, "home.html", Data)
		if err != nil {
			WriteLog("Error in Home execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "st-session", Value: ""})
	http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
}

func SelectPBX(w http.ResponseWriter, r *http.Request) {
	pbx := r.FormValue("pbx")
	if pbx != "" && pbx != "--Select PBX--" {
		http.SetCookie(w, &http.Cookie{
			Name:   "file",
			Value:  pbx,
			MaxAge: 60 * 60 * 24 * 30,
		})
		http.Redirect(w, r, "Status", http.StatusTemporaryRedirect)
	} else {
		http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
	}
}

type PBXType struct {
	HeaderType
	Count        int
	Page         string
	Title        string
	File         string
	Url          string
	AMIUser      string
	AMIPass      string
	Submit       string
	RemoteConfig string
	Protocol     string
}

func getPBXData(r *http.Request, Data *PBXType) {
	Data.Count, _ = strconv.Atoi(r.FormValue("index"))
	Data.Title = r.FormValue("title")
	Data.File = r.FormValue("file")
	Data.Url = r.FormValue("url")
	Data.AMIUser = r.FormValue("amiuser")
	Data.AMIPass = r.FormValue("amipass")
	Data.RemoteConfig = r.FormValue("remoteconfig")
	Data.Protocol = r.FormValue("protocol")
	if Data.Count == 0 {
		Data.Count = len(GetPBXFiles()) + 1
	}
}

func getPBXDefualt(Data *PBXType) {

	if Data.AMIUser == "" {
		Data.AMIUser = "admin"
	}
	if Data.Url == "" {
		Data.Url = "http://localhost:9091/"
	}
}

func PBXEmpty(Data *PBXType) (empty bool) {
	empty = strings.TrimSpace(Data.Title) == "" || strings.TrimSpace(Data.Url) == "" || strings.TrimSpace(Data.File) == ""
	if empty {
		Data.Message = "Empty parameter"
		Data.MessageType = "errormessage"
	}
	return
}

func AddPBX(w http.ResponseWriter, r *http.Request) {

	exist, User := CheckSession(r)
	if exist {
		if User.Admin {
			var Data PBXType
			Data.Submit = "Add"
			Data.Page = "Add new PBX for administration"
			Data.HeaderType = GetHeader(User, "AddPBX", r)
			getPBXData(r, &Data)
			if r.FormValue("add") != "" {
				if !PBXEmpty(&Data) {
					success := SavePbx(&Data, "")
					if success {
						WriteLog(User.Name + " Added: " + Data.File)
						http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
					}
				}
			} else {
				getPBXDefualt(&Data)
			}
			err := mytemplate.ExecuteTemplate(w, "pbx.html", Data)
			if err != nil {
				WriteLog("Error in AddPBX ExecuteTemplate: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "Login", http.StatusTemporaryRedirect)
	}
}

func EditPBX(w http.ResponseWriter, r *http.Request) {

	exist, User := CheckSession(r)
	if exist {
		if User.Admin {
			pbx := r.FormValue("pbx")
			pbxFile := GetPBXDir() + pbx
			if !FileExist(pbxFile) || pbx == "" {
				http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
			} else {
				var Data PBXType
				Data.Submit = "Update"
				Data.Page = "Edit PBX configuration"
				Data.HeaderType = GetHeader(User, "Home", r)
				getPBXData(r, &Data)
				if r.FormValue("add") != "" {
					if !PBXEmpty(&Data) {
						success := SavePbx(&Data, pbx)
						if success {
							if Data.RemoteConfig != "" {
								res, err := SaveRemoteFile(Data.Url, "/etc/simpletrunk/stagent.ini", Data.RemoteConfig)
								if err != nil {
									Data.Message = "Error: " + err.Error()
									Data.MessageType = "errormessage"
								} else {
									success = res.Success
									if !success {
										Data.Message = "Unable to write configuration: " + res.Message
										Data.MessageType = "errormessage"
									}
								}
							}
							if success {
								WriteLog(User.Name + " Edited: " + Data.File)
								http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
							}
						}
					}
				} else if r.FormValue("remove") != "" {
					filename := r.FormValue("filetoremove")
					if filename != pbx || strings.ContainsAny(filename, "/") {
						Data.Message = "Wrong File"
						Data.MessageType = "errormessage"
					} else {
						err := os.Rename(pbxFile, pbxFile+".bk")
						if err == nil {
							WriteLog(User.Name + " Removed: " + Data.File)
							http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
						}
					}
				} else {
					Data.Count, _ = strconv.Atoi(GetConfigValueFrom(pbxFile, "index", r.FormValue("index")))
					Data.Title = GetConfigValueFrom(pbxFile, "title", r.FormValue("title"))
					Data.File = r.FormValue("file")
					Data.Protocol = GetConfigValueFrom(pbxFile, "protocol", "sip")
					if Data.File == "" {
						Data.File = pbx
					}
					Data.Url = GetConfigValueFrom(pbxFile, "url", r.FormValue("url"))
					Data.AMIUser = GetConfigValueFrom(pbxFile, "amiuser", r.FormValue("amiuser"))
					Data.AMIPass = GetConfigValueFrom(pbxFile, "amipass", r.FormValue("amipass"))
					var err error
					Data.RemoteConfig, err = GetRemoteFile(Data.Url)
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					} else if strings.TrimSpace(Data.RemoteConfig) == "" {
						Data.RemoteConfig = "amiurl=http://localhost:8088/rawman\n" +
							"cdrdbserver=\n" +
							"cdrdatabase=\n" +
							"cdruser=\n" +
							"cdrpass=\n" +
							"cdrtable=\n" +
							"cdrkeyfield="
					}
					if Data.Count == 0 {
						Data.Count = len(GetPBXFiles()) + 1
					}
					getPBXDefualt(&Data)
				}
				err := mytemplate.ExecuteTemplate(w, "pbx.html", Data)
				if err != nil {
					WriteLog("Error in executeTemplate pbx.html edit: " + err.Error())
				}
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "Login", http.StatusTemporaryRedirect)
	}
}
