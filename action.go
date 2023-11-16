package main

import (
	"fmt"
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
	Data.HeaderType = GetHeader("", "", r)
	Data.ShowPages = false
	Data.Create = len(CallGetUsers()) == 0
	if r.FormValue("log") != "" {
		Data.Login = r.FormValue("login")
		Data.Password = r.FormValue("pass")
		Data.RememberMe = r.FormValue("rememberme") != ""
		if Data.Create {
			_, success, message := AddUser(Data.Login, Data.Password)
			if !success {
				Data.Message = message
				Data.MessageType = "errormessage"
			} else {
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
					cookie.MaxAge = 60 * 60 * 24 * 7
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

func CheckSession(r *http.Request) (exist bool, User UserType) {
	session, err := r.Cookie("st-session")
	exist = true
	if err != nil {
		exist = false
		WriteLog("Error in GetSession: " + err.Error())
	} else {
		User, exist = CallGetSession(session.Value)
	}
	return
}

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
		Data.HeaderType = GetHeader(User.Name, "Home", r)
		Data.Files = GetPBXFiles()
		err := mytemplate.ExecuteTemplate(w, "home.html", Data)
		if err != nil {
			WriteLog("Error in Home execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

var AdvancedTabs = []TabType{
	{Value: "Status", Name: "Status"},
	{Value: "Files", Name: "Files"},
	{Value: "SIPNodes", Name: "SIP"},
	{Value: "Dialplan", Name: "Dial plans"},
	{Value: "Commands", Name: "CLI commands"},
	{Value: "AMI", Name: "AMI commands"},
	{Value: "Terminal", Name: "Terminal"},
	{Value: "Logs", Name: "Logs"},
	{Value: "Config", Name: "Configuration"},
}

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
		Data.HeaderType = GetHeader(User.Name, "My Admin", r)
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
			co, _ := r.Cookie("reverse")
			co.MaxAge = 60 * 60 * 24 * 7
			co.Value = reverse
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

func PBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Header := GetHeader(User.Name, "PBX", r)
		err := mytemplate.ExecuteTemplate(w, "home.html", Header)
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
	if pbx != "" {
		http.SetCookie(w, &http.Cookie{
			Name:  "file",
			Value: pbx,
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
}

func getPBXData(r *http.Request, Data *PBXType) {
	Data.Count, _ = strconv.Atoi(r.FormValue("index"))
	Data.Title = r.FormValue("title")
	Data.File = r.FormValue("file")
	Data.Url = r.FormValue("url")
	Data.AMIUser = r.FormValue("amiuser")
	Data.AMIPass = r.FormValue("amipass")
	Data.RemoteConfig = r.FormValue("remoteconfig")
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
		var Data PBXType
		Data.Submit = "Add"
		Data.Page = "Add new PBX for administration"
		Data.HeaderType = GetHeader(User.Name, "AddPBX", r)
		getPBXData(r, &Data)
		if r.FormValue("add") != "" {
			if !PBXEmpty(&Data) {
				success := SavePbx(&Data)
				if success {
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
		http.Redirect(w, r, "Login", http.StatusTemporaryRedirect)
	}
}

func EditPBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := r.FormValue("pbx")
		pbxFile := GetPBXDir() + pbx
		if !FileExist(pbxFile) || pbx == "" {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		} else {
			var Data PBXType
			Data.Submit = "Update"
			Data.Page = "Edit PBX configuration"
			Data.HeaderType = GetHeader(User.Name, "Home", r)
			getPBXData(r, &Data)
			if r.FormValue("add") != "" {
				if !PBXEmpty(&Data) {
					os.Remove(pbxFile)
					success := SavePbx(&Data)
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
							if success {
								http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
							}
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
						http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
					}
				}
			} else {
				Data.Count, _ = strconv.Atoi(GetConfigValueFrom(pbxFile, "index", r.FormValue("index")))
				Data.Title = GetConfigValueFrom(pbxFile, "title", r.FormValue("title"))
				Data.File = r.FormValue("file")
				if Data.File == "" {
					Data.File = pbx
				}
				Data.Url = GetConfigValueFrom(pbxFile, "url", r.FormValue("url"))
				Data.AMIUser = GetConfigValueFrom(pbxFile, "amiuser", r.FormValue("amiuser"))
				Data.AMIPass = GetConfigValueFrom(pbxFile, "amipass", r.FormValue("amipass"))
				var err error
				Data.RemoteConfig, err = GetRemoteFile(Data.Url, Data.File)
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
		http.Redirect(w, r, "Login", http.StatusTemporaryRedirect)
	}
}

func GetAdvancedHeader(name, page, page2 string, r *http.Request) (Data HeaderType) {
	Data = GetHeader(name, "Advanced", r)
	AdvancedTabs := TabsType{Selected: page, Tabs: AdvancedTabs}
	Data.Tabs = append(Data.Tabs, AdvancedTabs)
	var Tabs TabsType
	switch page {
	case "Status":
		Tabs = TabsType{Selected: page2, Text: "Status",
			Tabs: []TabType{
				{Value: "?command=channels", Name: "Channels"},
				{Value: "?command=peers", Name: "Peers"},
				{Value: "?command=users", Name: "Users"},
				{Value: "?command=stats", Name: "Channel stats."},
				{Value: "?command=queues", Name: "Queue"},
				{Value: "?command=codecs", Name: "Codecs"},
			},
		}
	case "Files":
		Tabs = TabsType{Selected: page2, Text: "Files",
			Tabs: []TabType{
				{Value: "?file=asterisk.conf", Name: "asterisk.conf"},
				{Value: "?file=sip.conf", Name: "sip.conf"},
				{Value: "?file=extensions.conf", Name: "extensions.conf"},
				{Value: "?file=queues.conf", Name: "queues.conf"},
				{Value: "?file=agents.conf", Name: "agents.conf"},
				{Value: "?file=rtp.conf", Name: "rtp.conf"},
				{Value: "?file=cdr.conf", Name: "cdr.conf"},
				{Value: "?file=cdr_custom.conf", Name: "cdr_custom.conf"},
				{Value: "?file=manager.conf", Name: "manager.conf"},
				{Value: "?file=http.conf", Name: "http.conf"},
				{Value: "?file=all", Name: "All Files"},
			},
		}
	}
	if Tabs.Text != "" {
		Data.Tabs = append(Data.Tabs, Tabs)
	}
	return
}

func Advanced(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Data := GetAdvancedHeader(User.Name, "Advanced", "", r)
		err := mytemplate.ExecuteTemplate(w, "advanced.html", Data)
		if err != nil {
			WriteLog("Error in Advanced execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type StatusType struct {
	HeaderType
	Command string
	Content string
}

func Status(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	pbx := GetPBXDir() + GetCookieValue(r, "file")
	if exist {
		if FileExist(pbx) {
			var Data StatusType
			command := r.FormValue("command")
			var commandName, commandLine string
			switch command {
			case "channels":
				commandLine = "core show channels verbose"
			case "peers":
				commandLine = "sip show peers"
			case "users":
				commandLine = "sip show users"
			case "codecs":
				commandLine = "core show codecs"
			case "stats":
				commandLine = "sip show channelstats"
				commandName = "Channel stats."
			case "queues":
				commandLine = "queue show"
				commandName = "Queue"
			}
			if commandName == "" && command != "" {
				commandName = strings.ToUpper(string(command[0])) + command[1:len(command)]
			}
			Data.Command = command
			if commandLine != "" {
				Result, err := callAMICommand(pbx, commandLine)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				}
				if Result.Success {
					Data.Content = Result.Message
				} else {
					Data.Message = "Error: " + Result.Message
					Data.MessageType = "errormessage"
				}
			}
			Data.HeaderType = GetAdvancedHeader(User.Name, "Status", commandName, r)
			err := mytemplate.ExecuteTemplate(w, "status.html", Data)
			if err != nil {
				WriteLog("Error in Status execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type FilesType struct {
	HeaderType
	FileDataType
}

func Files(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	pbx := GetPBXDir() + GetCookieValue(r, "file")
	if exist {
		if FileExist(pbx) {
			var Data FilesType
			fileName := r.FormValue("file")
			if fileName == "" {
				fmt.Println(r.Form, r.PostForm)
				fileName = r.PostFormValue("file")
			}
			Data.HeaderType = GetAdvancedHeader(User.Name, "Files", fileName, r)
			var err error
			Data.FileDataType, err = GetFileData(fileName, pbx)
			if err != nil {
				Data.Message = "Error: " + err.Error()
				Data.MessageType = "errormessage"
			}
			err = mytemplate.ExecuteTemplate(w, "files.html", Data)
			if err != nil {
				WriteLog("Error in Files ExecuteTemplate: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func BackupFiles(w http.ResponseWriter, r *http.Request) {
}
