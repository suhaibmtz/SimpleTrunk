package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

func GetAdvancedHeader(name, page, page2 string, r *http.Request) (Data HeaderType) {
	Data = GetHeader(name, "Advanced", r)
	AdvancedTabs := TabsType{Selected: page, Tabs: AdvancedTabs}
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
			Data.HeaderType = GetAdvancedHeader(User.Name, "Status", commandName, r)
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
			NewUrl := strings.ReplaceAll(r.URL.String(), ";", "%3B")
			NewUrl = strings.ReplaceAll(NewUrl, "/SimpleTrunk/Files?", "")
			if fileName == "" {
				r.Form, _ = url.ParseQuery(NewUrl)
				fileName = r.FormValue("file")
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
		if FileExist(pbxfile) {
			var Data BackupFilesType
			Data.HeaderType = GetAdvancedHeader(User.Name, "BackupFiles", "", r)
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		if FileExist(pbxfile) {
			var Data CompareFilesType
			Data.HeaderType = GetAdvancedHeader(User.Name, "Comapre", "", r)
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data EditFileType
			Data.HeaderType = GetAdvancedHeader(User.Name, "EditFile", "", r)
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
						Data.Message = "Error: " + err.Error()
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
	Nodes []string
}

func SIPNodes(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data SipNodesType
			Data.HeaderType = GetAdvancedHeader(User.Name, "SIP", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}

			Res, err := GetFile(AgentUrl, "sip.conf")
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data EditNodeType
			Data.HeaderType = GetAdvancedHeader(User.Name, tabName, "", r)

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

			if Data.NodeName != "" {
				if r.FormValue("add") != "" {
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
				Data.Edit = r.FormValue("edit") != ""
				res, err := SaveNode(r, fileName, Data.NodeName, AgentUrl)
				var message string
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data DialplanType
			Data.HeaderType = GetAdvancedHeader(User.Name, "Dial plans", "", r)
			AgentUrl := GetConfigValueFrom(pbxfile, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			res, err := GetFile(AgentUrl, "extensions.conf")
			var message string
			if err != nil {
				message = err.Error()
			} else if !res.Success {
				message = res.Message
			}
			if message != "" {
				Data.Message = "Error: " + message
			}

			nodes := GetNodes(res.Content)
			for i, node := range nodes {
				var record TableListType
				record.Name = node
				record.NewTR = (i+1)%6 == 0
				Data.Nodes = append(Data.Nodes, record)
			}

			err = mytemplate.ExecuteTemplate(w, "advDialPlans.html", Data)
			if err != nil {
				WriteLog("Error in DialPlans execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
	if exist {
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
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
				commandLine = "sip reload"
				selected = "sip reload"
			case "dialplanreload":
				commandLine = "dialplan reload"
				selected = "dialplan reload"
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
			Data.HeaderType = GetAdvancedHeader(User.Name, "CLI commands", selected, r)
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
			err := mytemplate.ExecuteTemplate(w, "commands.html", Data)
			if err != nil {
				WriteLog("Error in commands execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data CommandType
			Data.HeaderType = GetAdvancedHeader(User.Name, "AMI commands", "", r)
			Data.Command = r.FormValue("command")
			if r.FormValue("execute") != "" {
				result, err := callAMI(pbxfile, Data.Command)
				if err != nil {
					Data.Message = "Error: " + err.Error()
					Data.MessageType = "errormessage"
				} else {
					if result.Success {
						if result.Message != "" {
							Data.Result = time.Now().String() + "\n" + result.Message
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func Terminal(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data CommandType
			Data.HeaderType = GetAdvancedHeader(User.Name, "Terminal", "", r)
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
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
		pbxfile := GetPBXDir() + GetCookieValue(r, "file")
		if FileExist(pbxfile) {
			var Data LogsType
			Data.HeaderType = GetAdvancedHeader(User.Name, "Logs", "", r)
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
			http.Redirect(w, r, "Home", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
