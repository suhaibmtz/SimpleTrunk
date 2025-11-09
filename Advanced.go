package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/motaz/codeutils"
)

var AdvancedTabs1 = []TabType{
	{Value: "Status", Name: "Status"},
	{Value: "Files", Name: "Files"},
	{Value: "SIPNodes", Name: "SIP"},
	{Value: "Dialplan", Name: "Dial plans"},
}
var AdvancedTabsAdmin = []TabType{
	{Value: "Commands", Name: "CLI commands"},
	{Value: "AMI", Name: "AMI commands"},
	{Value: "Terminal", Name: "Terminal"},
}
var AdvancedTabs2 = []TabType{
	{Value: "Logs", Name: "Logs"},
	{Value: "Config", Name: "Configuration"},
}

func GetSelectedPBX(r *http.Request) string {

	pbxname := GetCookieValue(r, "file")
	pbx := GetPBXDir() + pbxname
	return pbx
}

func GetSIPProtocol(r *http.Request) string {

	pbx := GetSelectedPBX(r)
	sip := GetConfigValueFrom(pbx, "protocol", "sip")
	return sip
}

func GetAdvancedHeader(User UserType, page, page2 string, r *http.Request) (Data HeaderType) {

	Data = GetHeader(User, "Advanced", r)
	AdvTabs := AdvancedTabs1
	if User.Admin {
		AdvTabs = append(AdvTabs, AdvancedTabsAdmin...)
	}
	AdvTabs = append(AdvTabs, AdvancedTabs2...)
	AdvancedTabs := TabsType{Selected: page, Tabs: AdvTabs}

	sip := GetSIPProtocol(r)

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
				{Value: "?file=" + sip + ".conf", Name: sip + ".conf"},
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
	case "EditFile":
		AdvancedTabs.Selected = "Files"
	case "CLI commands":
		Tabs = TabsType{Selected: page2, Text: "CLI Commands",
			Tabs: []TabType{
				{Value: "?command=corereload", Name: "core reload"},
				{Value: "?command=sipreload", Name: "sip reload"},
				{Value: "?command=dialplanreload", Name: "dialplan reload"},
				{Value: "?command=version", Name: "version"},
				{Value: "?command=help", Name: "Help"},
			},
		}

	}
	Data.Tabs = append(Data.Tabs, AdvancedTabs)
	if Tabs.Text != "" {
		Data.Tabs = append(Data.Tabs, Tabs)
	}
	return
}

func Advanced(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Data := GetAdvancedHeader(User, "Advanced", "", r)
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
	pbxname := GetCookieValue(r, "file")
	pbx := GetPBXDir() + pbxname
	sip := GetSIPProtocol(r)

	if exist {
		if FileExist(pbx) && pbxname != "" {
			var Data StatusType
			command := r.FormValue("command")
			var commandName, commandLine string
			switch command {
			case "channels":
				commandLine = "core show channels verbose"
			case "peers":
				if sip == "sip" {
					commandLine = "sip show peers"
				} else {
					commandLine = "pjsip show endpoints"
				}
			case "users":
				if sip == "sip" {
					commandLine = "sip show users"
				} else {
					commandLine = "pjsip show aors"
				}
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
			Data.HeaderType = GetAdvancedHeader(User, "Status", commandName, r)
			if commandLine != "" {
				Result, err := callAMICommand(pbx, commandLine)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				} else {
					if Result.Success {
						Data.Content = Result.Message
					} else {
						Data.Content = Result.Message
					}
				}
			}
			err := mytemplate.ExecuteTemplate(w, "status.html", Data)
			if err != nil {
				WriteLog("Error in Status execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
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
	pbxname := GetCookieValue(r, "file")
	pbx := GetPBXDir() + pbxname
	if exist {
		if FileExist(pbx) && pbxname != "" {
			var Data FilesType
			fileName := r.FormValue("file")
			NewUrl := strings.ReplaceAll(r.URL.String(), ";", "%3B")
			NewUrl = strings.ReplaceAll(NewUrl, "/SimpleTrunk/Files?", "")
			if fileName == "" {
				r.Form, _ = url.ParseQuery(NewUrl)
				fileName = r.FormValue("file")
			}
			Data.HeaderType = GetAdvancedHeader(User, "Files", fileName, r)
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
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type BackupFilesType struct {
	HeaderType
	ListFiles bool
	FileName  string
	Files     []string
	//backupfile
	OrignalFile string
	BackupFileContentType
}

func BackupFiles(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data BackupFilesType
			Data.HeaderType = GetAdvancedHeader(User, "BackupFiles", "", r)
			var obj = map[string]string{}
			fileName := r.FormValue("file")
			backupFileName := r.FormValue("backupfile")
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			if fileName != "" {
				obj["foldername"] = "/etc/asterisk/backup/"
				bytes, err := json.Marshal(obj)
				if err != nil {
					WriteLog("Error in BackupFiles Marshal obj listFiles: " + err.Error())
				}
				Data.Files, err = GetBackupFilesList(AgentUrl, bytes, fileName)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				}
				Data.ListFiles = true
				Data.FileName = fileName
			} else if backupFileName != "" {
				Data.FileName = backupFileName
				Data.OrignalFile = backupFileName[0 : strings.Index(backupFileName, "conf")+4]
				err, ret := doRetrieve(r, Data.OrignalFile, AgentUrl)
				if ret {
					if err == nil {
						WriteLog(User.Name + " Retrieved: " + Data.OrignalFile + " from " + backupFileName)
						Data.Message = "File Replaced"
						Data.MessageType = "infomessage"
					} else {
						Data.Message = err.Error()
						Data.MessageType = "errormessage"
					}
				}
				obj["filename"] = "/etc/asterisk/backup/" + backupFileName
				bytes, err := json.Marshal(obj)
				if err != nil {
					WriteLog("Error in BackupFiles Marshal obj BackUp file: " + err.Error())
				}
				Data.BackupFileContentType, err = GetBackupFileContents(AgentUrl, bytes, Data.OrignalFile, backupFileName)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				}
			} else {
				http.Redirect(w, r, "Files", http.StatusTemporaryRedirect)
			}
			err := mytemplate.ExecuteTemplate(w, "BackUpFiles.html", Data)
			if err != nil {
				WriteLog("Error in Advanced execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type LineType struct {
	LineN     int
	Line      string
	Color     string
	Span      string
	SpanColor string
}

type CompareFilesType struct {
	HeaderType
	Original    string
	BackUp      string
	OrgLines    []LineType
	BackUpLines []LineType
}

type DiffPosition struct {
	Type              string
	FirstFileStartPos int
	FirstFileEndPos   int

	SecondFileStartPos int
	SecondFileEndPos   int
}

func CompareFiles(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {

		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data CompareFilesType
			Data.HeaderType = GetAdvancedHeader(User, "Comapre", "", r)
			r.ParseForm()
			originalFileName := r.FormValue("originalfilename")
			backupFileName := r.FormValue("backupfilename")
			if r.FormValue("CompareFiles") != "" && originalFileName != "" && backupFileName != "" {
				url := GetConfigValueFrom(pbxfile, "url", "")
				Org, err := GetFileContents(url, "/etc/asterisk/"+originalFileName)
				if err != nil {
					WriteLog("Error in CompareFiles GetFileContent Original: " + err.Error())
				}
				Back, _ := GetFileContents(url, "/etc/asterisk/backup/"+backupFileName)

				command := "diff -w -b " + "/etc/asterisk/backup/" + backupFileName + "  /etc/asterisk/" + originalFileName
				obj := make(map[string]string)
				obj["command"] = command
				bytes, err := json.Marshal(obj)
				if err != nil {
					WriteLog("Error in CompareFiles marshal diff object: " + err.Error())
				} else {
					bytes, err = restCallURL(url+"Shell", bytes)
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					}
					var res ResponseType
					err = json.Unmarshal(bytes, &res)
					if err != nil {
						WriteLog("Error in CompareFiles Unmarshal Response: " + err.Error())
					}
					dpArr := diff(res)
					Data.Original = originalFileName
					Data.BackUp = backupFileName
					Data.OrgLines, Data.BackUpLines = CompareFile(Org, Back, originalFileName, backupFileName, dpArr)
				}
			} else {
				http.Redirect(w, r, "Files", http.StatusTemporaryRedirect)
			}
			err := mytemplate.ExecuteTemplate(w, "Compare.html", Data)
			if err != nil {
				WriteLog("Error in Advanced execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type EditFileType struct {
	HeaderType
	FileName string
	Content  string
}

func EditFile(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" && User.Admin {
			var Data EditFileType
			Data.HeaderType = GetAdvancedHeader(User, "EditFile", "", r)
			Data.FileName = r.FormValue("filename")
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			Response, err := GetFile(AgentUrl, Data.FileName)
			if err == nil {
				Data.Content = Response.Content
			} else {
				Data.Message = err.Error()
				Data.MessageType = "errormessage"
			}
			if r.FormValue("save") != "" {
				res, err := SaveRemoteFile(AgentUrl, Data.FileName, r.FormValue("content"))
				if err == nil {
					if res.Success {
						Data.Message = "File Saved"
						Data.MessageType = "infomessage"
					} else {
						Data.Message = "Error: " + res.Message
						Data.MessageType = "errormessage"
					}
				} else {
					Data.Message = err.Error()
					Data.MessageType = "errormessage"
				}
			}
			err = mytemplate.ExecuteTemplate(w, "EditFile.html", Data)
			if err != nil {
				WriteLog("Error in EditFile execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Files", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type SipNodesType struct {
	HeaderType
	Sip   string
	Nodes []string
}

func SIPNodes(w http.ResponseWriter, r *http.Request) {

	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx

		if FileExist(pbxfile) && pbx != "" {
			var Data SipNodesType
			Data.HeaderType = GetAdvancedHeader(User, "SIP", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			sip := GetSIPProtocol(r)

			Res, err := GetFile(AgentUrl, sip+".conf")
			Data.Sip = sip
			if err != nil {
				Data.Message = "Error: " + err.Error()
				Data.MessageType = "errormessage"
			} else {
				if Res.Success {
					nodes := GetNodes(Res.Content)
					reverseStr := GetCookieValue(r, "reverse")
					reverse := reverseStr == "yes"
					if reverse {
						for i := len(nodes) - 1; i >= 0; i-- {
							Data.Nodes = append(Data.Nodes, nodes[i])
						}
					} else {
						Data.Nodes = nodes
					}

				}
			}
			err = mytemplate.ExecuteTemplate(w, "sipnodes.html", Data)
			if err != nil {
				WriteLog("Error in sipNodes execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type EditNodeType struct {
	HeaderType
	FileName string
	NodeName string
	Content  string
	Command  string
	Caption  string
	Edit     bool
}

func EditNode(w http.ResponseWriter, r *http.Request) {

	exist, User := CheckSession(r)
	if exist {
		tabName := "SIP"
		fileName := r.FormValue("filename")
		if !strings.Contains(fileName, "sip.") {
			tabName = "Dial plans"
		}
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data EditNodeType
			Data.HeaderType = GetAdvancedHeader(User, tabName, "", r)

			Data.FileName = fileName
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}

			Data.NodeName = r.FormValue("nodename")
			if !strings.Contains(Data.NodeName, "[") && Data.NodeName != "" {
				Data.NodeName = "[" + Data.NodeName + "]"
			}
			if Data.NodeName == "" && !User.Admin {
				page := "SIPNodes"
				if tabName == "Dial plans" {
					page = "Dialplan"
				}
				http.Redirect(w, r, page, http.StatusTemporaryRedirect)
			}

			if Data.NodeName != "" {
				if r.FormValue("add") != "" && User.Admin {
					message := addNewNode(fileName, Data.NodeName, r.FormValue("content"), AgentUrl)
					if message == "" {
						WriteLog(User.Name + " Added: " + Data.NodeName)
						Data.Message = "New node " + Data.NodeName + " has been added"
						Data.MessageType = "infomessage"
					} else {
						Data.Message = "Error: " + message
						Data.MessageType = "errormessage"
					}
				}
				Data.Edit = r.FormValue("edit") != "" && User.Admin
				var message string
				if Data.IsAdmin {
					res, err := SaveNode(r, fileName, Data.NodeName, AgentUrl)
					if err != nil {
						message = err.Error()
					} else if !res.Success {
						message = res.Message
					} else {
						Data.Message = "Saved"
						Data.MessageType = "infomessage"
						Data.Command, Data.Caption = GetReloadCommand(fileName)
					}

					if message != "" {
						Data.Message = message
						Data.MessageType = "errormessage"
					}
				}

				Data.Content, message = GetNodeContent(fileName, AgentUrl, Data.NodeName)
				if message != "" {
					Data.Message = message
					Data.MessageType = "errormessage"
				}
			}

			err := mytemplate.ExecuteTemplate(w, "editnode.html", Data)
			if err != nil {
				WriteLog("Error in EditNode execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type DialplanType struct {
	HeaderType
	Nodes []TableListType
}

func Dialplan(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data DialplanType
			Data.HeaderType = GetAdvancedHeader(User, "Dial plans", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var err error
			Data.Nodes, err = GetDialplans(AgentUrl)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			err = mytemplate.ExecuteTemplate(w, "advDialPlans.html", Data)
			if err != nil {
				WriteLog("Error in DialPlans execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}

}

type CommandsType struct {
	HeaderType
	TextCommand bool
	Command     string
	Result      string
}

func Commands(w http.ResponseWriter, r *http.Request) {

	exist, User := CheckSession(r)
	sip := GetSIPProtocol(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			if User.Admin {
				var Data CommandsType
				AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
				if AgentUrl != "" {
					if string(AgentUrl[len(AgentUrl)-1]) != "/" {
						AgentUrl += "/"
					}
				}
				command := r.FormValue("command")
				Data.Command = r.FormValue("commandtext")
				var commandLine string
				var selected string
				switch command {
				case "corereload":
					commandLine = "core reload"
					selected = "core reload"
				case "sipreload":
					commandLine = sip + " reload"
					selected = commandLine
				case "dialplanreload":
					commandLine = "dialplan reload"
					selected = commandLine
				case "version":
					commandLine = "core show version"
				case "help":
					commandLine = "core show help"
				case "text":
					commandLine = Data.Command
				}
				if selected == "" {
					selected = command
				}
				if selected != "text" {
					Data.Command = selected
				}
				Data.HeaderType = GetAdvancedHeader(User, "CLI commands", selected, r)
				if command != "" {
					Data.TextCommand = command == "text"

					var res ResponseType
					var err error
					if strings.Contains(commandLine, "reload") {
						res, err = callCLI(AgentUrl, commandLine)
					} else {
						res, err = callAMICommand(pbxfile, commandLine)
					}
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					}
					if res.Success {
						Data.Result = res.Message
					}
					if Data.Result == "" {
						Data.Result = res.Result
					}
				}
				if r.FormValue("ret") != "" {
					http.Redirect(w, r, r.Header.Get("referer"), http.StatusTemporaryRedirect)
				}
				err := mytemplate.ExecuteTemplate(w, "commands.html", Data)
				if err != nil {
					WriteLog("Error in commands execute template: " + err.Error())
				}
			} else {
				http.Redirect(w, r, "Advanced", http.StatusTemporaryRedirect)
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type CommandType struct {
	HeaderType
	Command string
	Result  string
}

func AMI(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			if User.Admin {
				var Data CommandType
				Data.HeaderType = GetAdvancedHeader(User, "AMI commands", "", r)
				Data.Command = r.FormValue("command")
				if r.FormValue("execute") != "" {
					result, err := callAMI(pbxfile, Data.Command)
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					} else {
						if result.Success {
							if result.Message != "" {
								Data.Result = time.Now().Format("Mon Jan 2 15:04:05 MST 2006") + "\n" + result.Message
							}
						} else {
							Data.Result = result.Message
						}
					}

				}
				err := mytemplate.ExecuteTemplate(w, "ami.html", Data)
				if err != nil {
					WriteLog("Error in commands execute template: " + err.Error())
				}
			} else {
				http.Redirect(w, r, "Advanced", http.StatusTemporaryRedirect)
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func Terminal(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			if User.Admin {
				var Data CommandType
				Data.HeaderType = GetAdvancedHeader(User, "Terminal", "", r)
				AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
				if AgentUrl != "" {
					if string(AgentUrl[len(AgentUrl)-1]) != "/" {
						AgentUrl += "/"
					}
				}
				Data.Command = r.FormValue("command")
				if r.FormValue("execute") != "" {
					res, err := executeShell(Data.Command, AgentUrl)
					if err != nil {
						Data.Message = err.Error()
						Data.MessageType = "errormessage"
					} else {
						if res.Success {
							Data.Result = res.Result
						} else {
							Data.Result = res.Message
						}
					}

				}
				err := mytemplate.ExecuteTemplate(w, "terminal.html", Data)
				if err != nil {
					WriteLog("Error in Terminal execute template: " + err.Error())
				}
			} else {
				http.Redirect(w, r, "Advanced", http.StatusTemporaryRedirect)
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type LogsType struct {
	HeaderType
	Result string
	File   string
	Lines  string
}

func Logs(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data LogsType
			Data.HeaderType = GetAdvancedHeader(User, "Logs", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			linesStr := r.FormValue("size")
			if linesStr == "" {
				linesStr = GetCookieValue(r, "logsize")
				if linesStr == "" {
					linesStr = "40"
				}
			}
			Data.File = r.FormValue("file")
			Data.Lines = linesStr
			if Data.File != "" {
				var fileName string
				if Data.File == "Messages" {
					fileName = "/var/log/asterisk/messages"
				} else if Data.File == "Full" {
					fileName = "/var/log/asterisk/full"

				}
				if !codeutils.IsFileExists(fileName) {
					fileName += ".log"
				}

				co := &http.Cookie{Name: "logsize", Value: linesStr}
				http.SetCookie(w, co)

				// Call service
				res, err := GetLogTail(AgentUrl, fileName, Data.Lines)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				} else {
					if res.Success {
						Data.Result = res.Content
					} else {
						Data.Message = res.Message
						Data.MessageType = "errormessage"
					}
				}
			}
			err := mytemplate.ExecuteTemplate(w, "logs.html", Data)
			if err != nil {
				WriteLog("Error in logs execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func Config(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			Data := GetAdvancedHeader(User, "Configuration", "", r)
			err := mytemplate.ExecuteTemplate(w, "config.html", Data)
			if err != nil {
				WriteLog("Error in Config execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func Backup(w http.ResponseWriter, r *http.Request) {
	exist, _ := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			WriteLog("Download file called from: " + r.RemoteAddr)
			w.Header().Set("Content-Type", "application/zip")

			obj := make(map[string]interface{})
			obj["directory"] = "/etc/asterisk/"
			obj["ext"] = ".conf"
			obj["name"] = pbxfile

			bytes, _ := json.Marshal(obj)
			WriteLog("Downloading: " + pbx)
			w.Header().Set("Content-Disposition", "attachment;filename="+pbx+".zip")

			op, err := DownloadFile(AgentUrl+"BackupFiles", bytes, "application/zip", w)
			if err != nil {
				WriteLog("Error downloading file: " + err.Error())
				return
			}

			WriteLog("Size: " + strconv.FormatInt(op.Size, 10))
			r.ContentLength = op.Size

		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type SoundFileType struct {
	IsDir bool
	Name  string
}

type UploadSoundType struct {
	HeaderType
	Dir    string
	Parent string
	Files  []SoundFileType
}

func UploadSound(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data UploadSoundType
			Data.HeaderType = GetAdvancedHeader(User, "Configuration", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			dir := r.FormValue("dir")
			if dir == "" {
				rdir := r.FormValue("rdir")
				if rdir != "" {
					dir = rdir
				} else {
					dir = "/usr/share/asterisk/sounds"
				}
			}
			message := r.FormValue("message")

			if message != "" {
				var obj ReciveFileResponseType
				err := json.Unmarshal([]byte(message), &obj)
				if err != nil {
					WriteLog("Error in UploadSound Unmarshal: " + err.Error())
				}
				filename := obj.FileName
				amessage := obj.Message
				if obj.Success {
					Data.MessageType = "infomessage"
					Data.Message = "File: " + filename + " : " + amessage
				} else {
					Data.MessageType = "warnmessage"
					Data.Message = "File: " + filename + " : " + amessage
				}
			}
			dir = addSlash(dir)
			directory := r.FormValue("directory")
			if directory != "" {
				dir = addSlash(dir + directory)
			}
			if len(dir) > 2 {
				parent := removeSlash(dir)
				parent = parent[0:strings.LastIndex(parent, string(os.PathSeparator))]
				if parent == "" {
					parent = string(os.PathSeparator)
				}
				Data.Parent = parent
			}
			Data.Dir = dir
			filesRes, err := listFiles(AgentUrl, dir)
			if err != nil {
				Data.Message = err.Error()
				Data.MessageType = "errormessage"
			} else {
				files := filesRes.Files
				for i := 0; i < len(files); i++ {
					var record SoundFileType
					record.Name = files[i]
					record.IsDir = !(strings.Index(record.Name, ".") > 0)
					Data.Files = append(Data.Files, record)
				}
			}
			err = mytemplate.ExecuteTemplate(w, "uploadsound.html", Data)
			if err != nil {
				WriteLog("Error in UploadSound execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func PlaySound(w http.ResponseWriter, r *http.Request) {
	exist, _ := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {

			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}

			contenttype := "audio/wav"
			filename := r.FormValue("filename")
			w.Header().Set("ContentType", contenttype)
			obj := make(map[string]string)
			obj["filename"] = filename
			obj["contenttype"] = contenttype
			bytes, _ := json.Marshal(obj)
			lastindex := strings.LastIndex(filename, "/")
			name := filename
			if lastindex != -1 {
				name = filename[lastindex+1 : len(filename)-1]
			}

			w.Header().Set("Content-Disposition", "attachment;filename="+name)
			DownloadFile(AgentUrl+"DownloadFile", bytes, contenttype, w)

		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type ReciveFileResponseType struct {
	ResponseType
	FileName string `json:"filename"`
}

func UploadSoundFile(w http.ResponseWriter, r *http.Request) {
	exist, _ := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {

			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			dir := r.FormValue("dir")
			uploadurl := AgentUrl + "ReceiveFile"
			r.ParseMultipartForm(20 << 40)
			file, handler, err := r.FormFile("file")
			var resp []byte
			if err != nil {
				WriteLog("Error in UploadSoundFile form File: " + err.Error())
			} else {
				jsonrequest := make(map[string]any)
				jsonrequest["filename"] = handler.Filename
				jsonrequest["dir"] = dir
				bytes, _ := io.ReadAll(file)
				var content []string
				for _, line := range strings.Split(string(bytes), "\n") {
					content = append(content, base64.StdEncoding.EncodeToString([]byte(line+"\n")))
				}
				jsonrequest["content"] = content
				data, _ := json.Marshal(jsonrequest)
				resp, err = restCallURL(uploadurl, data)
			}
			rmessage := string(resp)
			http.Redirect(w, r, "UploadSound?rdir="+dir+"&message="+rmessage, http.StatusTemporaryRedirect)
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}

}

type AMIConfigType struct {
	HeaderType
	// status and users
	Ami       bool
	Http      bool
	Success   bool
	Connected bool
	Users     []AMIUserType
	//edf
	Spl  []string
	User string
}

func AMIparamStatus(request *http.Request, admin bool) bool {
	return (request.FormValue("adf") == "" && request.FormValue("edf") == "") || !admin
}

func CDRparamStatus(request *http.Request, admin bool) bool {
	return (request.FormValue("cf") == "" && request.FormValue("edit") == "") || !admin
}

func setDefault(w http.ResponseWriter, Aurl, pbxfile string, r *http.Request, uname string) (err error) {
	var spl []string
	var user string
	obj := make(map[string]string)
	obj["Username"] = uname
	bytes, _ := json.Marshal(obj)
	var response []byte
	response, err = restCallURL(Aurl+"GetAMIUserInfo", bytes)
	if err == nil {
		var res ResponseType
		json.Unmarshal(response, &res)
		if res.Success {
			spl = strings.Split(res.Result, ":")
			user = strings.ReplaceAll(spl[0], "[", "")
			user = strings.ReplaceAll(user, "]", "")
			SetConfigValueTo(pbxfile, "amiuser", user)
			SetConfigValueTo(pbxfile, "amipass", spl[1])
			http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
		} else {
			err = errors.New(res.Message)
		}
	}
	return
}

func doAddAMIUser(r *http.Request, w http.ResponseWriter, Aurl string) (err error) {
	obj := make(map[string]string)
	obj["Username"] = r.FormValue("user")
	obj["Secret"] = r.FormValue("sec")
	obj["Read"] = r.FormValue("read")
	obj["Write"] = r.FormValue("write")
	obj["Addi"] = r.FormValue("addi")
	bytes, _ := json.Marshal(obj)
	var response []byte
	response, err = restCallURL(Aurl+"AddAMIUser", bytes)
	if err == nil {
		var res ResponseType
		json.Unmarshal(response, &res)
		if res.Success {
			http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
		} else {
			err = errors.New("Error in Adding AMI User: " + res.Message)
		}
	}
	return
}

func editAMIUserForm(Aurl, uname string) (err error, spl []string, user string) {
	obj := make(map[string]string)
	obj["Username"] = uname
	data, _ := json.Marshal(obj)
	var response []byte
	response, err = restCallURL(Aurl+"GetAMIUserInfo", data)
	if err == nil {
		var res ResponseType
		json.Unmarshal(response, &res)
		if res.Success {
			spl = strings.Split(res.Result, ":")
			user = strings.ReplaceAll(spl[0], "[", "")
			user = strings.ReplaceAll(user, "]", "")
		} else {
			err = errors.New(res.Message)
		}
	}
	return
}

func doModAMIUser(r *http.Request, w http.ResponseWriter, Aurl string) (err error) {
	obj := make(map[string]string)
	obj["Username"] = r.FormValue("cuser")
	obj["NUsername"] = r.FormValue("user")
	obj["Secret"] = r.FormValue("sec")
	obj["Read"] = r.FormValue("read")
	obj["Write"] = r.FormValue("write")
	obj["Addi"] = r.FormValue("addi")
	req, _ := json.Marshal(obj)
	var data []byte
	data, err = restCallURL(Aurl+"ModifyAMIUser", req)
	if err == nil {
		var res ResponseType
		json.Unmarshal(data, &res)
		if res.Success {
			r.Form = make(url.Values)
			http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
		} else {
			err = errors.New("Error in Adding AMI User: " + res.Message)
		}
	}
	return
}

func AMIConfig(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) && pbx != "" {
			var Data AMIConfigType
			Data.HeaderType = GetAdvancedHeader(User, "Configuration", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var err error
			if r.FormValue("def") != "" {
				if User.Admin {
					err = setDefault(w, AgentUrl, pbxfile, r, r.FormValue("def"))
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					}
				} else {
					http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
				}
			}
			if r.FormValue("adf") != "" {
				if User.Admin {
					if r.FormValue("aok") != "" {
						err = doAddAMIUser(r, w, AgentUrl)
						if err != nil {
							Data.Message = "Error: " + err.Error()
							Data.MessageType = "errormessage"
						}
					}
					err = mytemplate.ExecuteTemplate(w, "AMIConfigAdf.html", Data)
				} else {
					http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
				}
			}
			if r.FormValue("edf") != "" {
				if User.Admin {
					if r.FormValue("mok") != "" {
						err = doModAMIUser(r, w, AgentUrl)
						if err != nil {
							Data.Message = "Error: " + err.Error()
							Data.MessageType = "errormessage"
						}
					}
					err, Data.Spl, Data.User = editAMIUserForm(AgentUrl, r.FormValue("edf"))
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					}
					err = mytemplate.ExecuteTemplate(w, "AMIConfigEdf.html", Data)
				} else {
					http.Redirect(w, r, "AMIConfig", http.StatusTemporaryRedirect)
					return
				}
			}
			if AMIparamStatus(r, User.Admin) {
				Data.Users, Data.Success, _ = AMIUsers(pbxfile, AgentUrl)
				err, Data.Ami, Data.Http = AMIStatus(AgentUrl)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				}
				Data.Connected = err == nil
				err = mytemplate.ExecuteTemplate(w, "AMIConfig.html", Data)
			}
			if err != nil {
				WriteLog("Error in AMIConfig: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type AMIUserType struct {
	User    string
	Spl     []string
	Default bool
}

func getDefault(user, pass, pbxfile string) (res bool) {
	suser := GetConfigValueFrom(pbxfile, "amiuser", "")
	spass := GetConfigValueFrom(pbxfile, "amipass", "")
	if suser == user && spass == pass {
		res = true
	} else {
		res = false
	}

	return res
}

func AMIUsers(pbxfile, Aurl string) (users []AMIUserType, success bool, err error) {
	var bytes []byte
	bytes, err = restCallURL(Aurl+"GetAMIUsersInfo", nil)
	if err == nil {
		var res ResponseType
		err = json.Unmarshal(bytes, &res)
		if err == nil {
			if res.Success {
				if res.Result != "" {
					success = true
					spl := strings.Split(res.Result, ";")
					for i := 0; i+1 < len(spl); i++ {
						spl1 := strings.Split(spl[i], ":")
						user := strings.ReplaceAll(spl1[0], "[", "")
						user = strings.ReplaceAll(user, "]", "")
						users = append(users, AMIUserType{User: user, Spl: spl1, Default: getDefault(user, spl1[1], pbxfile)})
					}
				}
			} else {
				err = errors.New(res.Message)
			}
		}
	}
	return
}

func AMIStatus(Aurl string) (err error, ami /*amiht,*/, amihttp bool) {
	var spl []string
	var data []byte
	data, err = restCallURL(Aurl+"GetAMIStatus", nil)
	if err == nil {
		var res ResponseType
		json.Unmarshal(data, &res)
		if res.Success {
			spl = strings.Split(res.Result, ":")
			ami = spl[0] == "ok"
			//amiht = spl[1] == "ok"
			amihttp = spl[2] == "ok"
		} else {
			err = errors.New(res.Message)
		}
	}
	return
}

type CDRConfigType struct {
	HeaderType
	Connected  bool
	Success    bool
	CDRStatus  bool
	CDRMessage string
	Spl        []string
}

func checkCDRStatus(Aurl string) (res ResponseType, err error) {
	var data []byte
	data, err = restCallURL(Aurl+"GetCDRConfStatus", nil)
	if err == nil {
		json.Unmarshal(data, &res)
	}
	return
}

func GetCDRConf(Aurl string) (err error, spl []string) {
	var data []byte
	data, err = restCallURL(Aurl+"GetCDRConf", nil)
	if err == nil {
		var res ResponseType
		json.Unmarshal(data, &res)
		if res.Success {
			spl = strings.Split(res.Result, ":")
		} else {
			err = errors.New("Error in CDR configuration: " + res.Message)
		}
	}
	return
}

func GetCDRStatus(Aurl string) (err error, success bool, message string) {
	var res ResponseType
	res, err = checkCDRStatus(Aurl)
	success = res.Success
	message = res.Message
	return
}

func EditCDRConfigForm(Aurl string) (err error, spl []string) {
	var data []byte
	data, err = restCallURL(Aurl+"GetCDRConf", nil)
	if err == nil {
		var res ResponseType
		json.Unmarshal(data, &res)
		if res.Success {
			spl = strings.Split(res.Result, ":")
		} else {
			err = errors.New("Error in CDR configuration: " + res.Result)
		}
	}
	return
}

func CDRConfig(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	pbxname := GetCookieValue(r, "file")
	if exist {
		pbx := GetPBXDir() + pbxname
		if FileExist(pbx) && pbxname != "" {
			var Data CDRConfigType
			Data.HeaderType = GetAdvancedHeader(User, "Configuration", "", r)
			AgentUrl := GetConfigValueFrom(pbx, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var err error
			var chres ResponseType
			var checkRes []byte
			checkRes, err = restCallURL(AgentUrl+"IsCDRConf", nil)
			if err != nil {
				Data.Message = err.Error()
				Data.MessageType = "errormessage"
			}
			Data.Connected = err == nil
			if CDRparamStatus(r, User.Admin) {
				json.Unmarshal(checkRes, &chres)
				chsuccess := chres.Success
				csres, _ := checkCDRStatus(AgentUrl)
				cssuccess := csres.Success
				if !chsuccess && !cssuccess {
					Data.Success = false
				}
				if chsuccess || cssuccess {
					Data.Success = true
					err, Data.Spl = GetCDRConf(AgentUrl)
					if err == nil {
						err, Data.CDRStatus, Data.CDRMessage = GetCDRStatus(AgentUrl)
					}
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					}
				}
				err = mytemplate.ExecuteTemplate(w, "CDRConfig.html", Data)
			}
			if r.FormValue("cf") != "" && User.Admin {
				if r.FormValue("cok") != "" {
					ser := r.FormValue("ser")
					dbuname := r.FormValue("duname")
					dbpass := r.FormValue("dpass")
					dname := r.FormValue("dname")
					cdrtab := r.FormValue("cdrtab")
					keyf := r.FormValue("key")
					obj := make(map[string]string)
					obj["Server"] = ser
					obj["Duname"] = dbuname
					obj["Dpass"] = dbpass
					obj["Dname"] = dname
					obj["Ctab"] = cdrtab
					obj["Ckey"] = keyf
					bytes, _ := json.Marshal(obj)
					var data []byte
					data, err = restCallURL(AgentUrl+"SetCDRConf", bytes)
					if err == nil {
						var res ResponseType
						json.Unmarshal(data, &res)
						if res.Success {
							http.Redirect(w, r, "CDRConfig", http.StatusTemporaryRedirect)
						} else {
							Data.Message = "Error in CDR configuration: " + res.Message
							Data.MessageType = "errormessage"
						}
					}
				}
				err = mytemplate.ExecuteTemplate(w, "CDRConfigCf.html", Data)
			}
			if r.FormValue("edit") != "" && User.Admin {
				if r.FormValue("mcok") != "" {
					ser := r.FormValue("ser")
					dbuname := r.FormValue("duname")
					dbpass := r.FormValue("dpass")
					dname := r.FormValue("dname")
					cdrtab := r.FormValue("cdrtab")
					keyf := r.FormValue("key")
					obj := make(map[string]string)
					obj["Server"] = ser
					obj["Duname"] = dbuname
					obj["Dpass"] = dbpass
					obj["Dname"] = dname
					obj["Ctab"] = cdrtab
					obj["Ckey"] = keyf
					bytes, _ := json.Marshal(obj)
					var data []byte
					data, err = restCallURL(AgentUrl+"ModifyCDRConf", bytes)
					if err == nil {
						var res ResponseType
						json.Unmarshal(data, &res)
						if res.Success {
							r.Form = make(url.Values)
							http.Redirect(w, r, "CDRConfig", http.StatusTemporaryRedirect)
						} else {
							err = errors.New("Error in CDR configuration: " + res.Message)
						}
					}
					if err != nil {
						Data.Message = err.Error()
						Data.MessageType = "errormessage"
					}
				}
				err, Data.Spl = EditCDRConfigForm(AgentUrl)
				if err != nil {
					Data.Message = err.Error()
					Data.MessageType = "errormessage"
				}
				err = mytemplate.ExecuteTemplate(w, "CDRConfigEdit.html", Data)
			}
			if err != nil {
				WriteLog("Error in CDRConfig execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
