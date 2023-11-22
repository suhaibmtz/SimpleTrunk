package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

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
				doRetrieve(r, Data.OrignalFile, AgentUrl)
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
				var obj = map[string]string{}

				obj["filename"] = "/etc/asterisk/" + originalFileName
				bytes, _ := json.Marshal(obj)
				Org, err := GetFileContents(url, obj["filename"], bytes)
				if err != nil {
					WriteLog("Error in CompareFiles GetFileContent Original: " + err.Error())
				}
				obj["filename"] = "/etc/asterisk/backup/" + backupFileName
				bytes, _ = json.Marshal(obj)

				Back, _ := GetFileContents(url, obj["filename"], bytes)

				command := "diff -w -b " + "/etc/asterisk/backup/" + backupFileName + "  /etc/asterisk/" + originalFileName
				obj["command"] = command
				bytes, err = json.Marshal(obj)
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
