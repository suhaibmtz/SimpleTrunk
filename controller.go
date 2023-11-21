package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type UserType struct {
	ID       int
	Name     string
	Password string
}

func GetPBXFiles() (Files []PBXFileType) {
	CheckFolder()
	InfoFiles := GetPBXFilesInfo()
	sorted := false
	PBXPath := GetPBXDir()
	for !sorted {
		sorted = true
		for i := 0; i < len(InfoFiles)-1; i++ {
			aStr := GetConfigValueFrom(PBXPath+InfoFiles[i].Name(), "index", "0")
			bStr := GetConfigValueFrom(PBXPath+InfoFiles[i+1].Name(), "index", "0")
			a, _ := strconv.Atoi(aStr)
			b, _ := strconv.Atoi(bStr)
			if a > b {
				InfoFiles[i], InfoFiles[i+1] = InfoFiles[i+1], InfoFiles[i]
				sorted = false
			}
		}
	}
	for i, file := range InfoFiles {
		record := GetPBXFile(file.Name())
		i1 := i + 1
		if i1%2 == 0 {
			record.Color = "#dAbaa7"
		} else {
			record.Color = "#dececa"
		}
		record.NewTR = i1%5 == 0
		Files = append(Files, record)
	}
	return
}

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

func GetHeader(username string, Tab string, r *http.Request) HeaderType {
	var Header HeaderType
	Header.LogoutText = username
	PBX, err := r.Cookie("file")
	if err == nil {
		Header.SelectedPBX = GetPBXFile(PBX.Value)
	}
	Header.Version = Version
	Header.MainTabs.Selected = Tab
	Header.MainTabs.Tabs = taps
	Header.PBXFiles = GetPBXFiles()
	Header.ShowPages = true
	return Header
}

func GetPBXFile(name string) (file PBXFileType) {
	CheckFolder()
	file.FileName = name
	file.Path = GetPBXDir() + name
	if FileExist(file.Path) && file.FileName != "" {
		indexStr := GetConfigValueFrom(file.Path, "index", "0")
		file.Index, _ = strconv.Atoi(indexStr)
		file.Title = GetConfigValueFrom(file.Path, "title", "")
		file.Url = GetConfigValueFrom(file.Path, "url", "")
		fileLength := len(file.FileName)
		file.IsStc = file.FileName[fileLength-4:fileLength] == ".stc"
		if len(file.Url) > 0 {
			file.IP = file.Url[strings.Index(file.Url, "//")+2 : len(file.Url)]
			if strings.Contains(file.IP, ":") {
				file.IP = file.IP[0:strings.Index(file.IP, ":")]
			} else {
				file.IP = file.IP[0:strings.Index(file.IP, "/")]
			}
			switch string(file.IP[len(file.IP)-1]) {
			case ":", "/", `\`:
				file.IP = file.IP[0 : len(file.IP)-1]
			}
		}
	}
	return
}

func GetUserBy(what, value string) (User UserType, exist bool) {
	CheckFolder()
	exist = true
	var err error
	User, err = GetUser(what, value)
	if err != nil {
		exist = false
		WriteLog("Error in GetUserBy: " + err.Error())
	}
	if User.ID == 0 {
		exist = false
	}
	return
}

func GetUserByID(id int) (User UserType, exist bool) {
	return GetUserBy("id", fmt.Sprint(id))
}

func GetUserByName(user string) (User UserType, exist bool) {
	return GetUserBy("name", user)
}

func CallGetUsers() (Users []UserType) {
	CheckFolder()
	var err error
	Users, err = GetUsers()
	if err != nil {
		WriteLog("Error in GetUsers: " + err.Error())
	}
	return
}

func AddUser(name, password string) (User UserType, success bool, message string) {
	success = false
	CheckFolder()
	var exist bool
	if name != "" {
		if password != "" {
			User, exist = GetUserByName(name)
			if !exist {
				err := InsertUser(name, GetMD5(password))
				if err != nil {
					message = "Error: " + err.Error()
					WriteLog("Error in AddUser: " + err.Error())
				} else {
					User, _ = GetUserByName(name)
					success = true
				}
			} else {
				message = "User already exist"
			}
		} else {
			message = "Empty password"
		}
	} else {
		message = "Empty username"
	}
	return
}

func AddTable(name, table string) {
	err := InsertTable(name, table)
	if err != nil {
		WriteLog("Error in CreateTable " + name + ": " + err.Error())
	}
}

var UsersTable = `(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name CHAR(25),
	password CHAR(30)
	)`

var SessionTable = `(
	key CHAR(30),
	id int
	)`

func CheckFolder() {
	PBXPath := GetPBXDir()
	CreatePBX := !FileExist(PBXPath)
	if !FileExist(simpletrunkPath) {
		CreateFolder(simpletrunkPath)
		connectDB()
		CreatePBX = true
	}
	if CreatePBX {
		CreateFolder(PBXPath)
	}
	AddTable("users", UsersTable)
	AddTable("session", SessionTable)
}

func CallGetSession(key string) (User UserType, exist bool) {
	exist = true
	id, err := GetSession(key)
	if err != nil {
		WriteLog("Error in GetSession: " + err.Error())
		exist = false
	}
	if exist {
		User, exist = GetUserByID(id)
	}
	return
}

func CallSetSession(key string, id int) {
	err := SetSession(key, id)
	if err != nil {
		WriteLog("Error in SetSession: " + err.Error())
	}
}

func GetRemoteFile(url, filename string) (text string, Callerr error) {

	var obj = map[string]string{}
	obj["filename"] = "/etc/simpletrunk/stagent.ini"
	data, err := json.Marshal(obj)
	if err != nil {
		WriteLog("Error in GetRemoteFile marshal obj: " + err.Error())
	}
	if url != "" {
		if string(url[len(url)-1]) != "/" {
			url += "/"
		}
	}
	var res []byte
	res, Callerr = restCallURL(url+"GetFile", data)
	if Callerr != nil {
		WriteLog("Error in GetRemoteFile restCallURL: " + Callerr.Error())
	}

	var Response GetFileResponseType
	err = json.Unmarshal(res, &Response)
	if err != nil {
		WriteLog("Error in GetRemoteFile Unmarshal: " + err.Error())
	}
	if Response.Success {
		text = Response.Content
	} else if err == nil {
		text = "Error: " + Response.Message
	}
	return
}

func SavePbx(Data *PBXType) (success bool) {
	CheckFolder()
	dir := GetPBXDir()
	if !strings.Contains(Data.File, ".") {
		Data.File += ".stc"
	}
	Data.File = dir + Data.File
	_, err := os.Create(Data.File)
	if os.IsExist(err) {
		Data.Message = "Already Exist"
		Data.MessageType = "errormessage"
	} else {
		Data.Url = strings.ReplaceAll(Data.Url, `\`, "")
		success = SetConfigValueTo(Data.File, "url", Data.Url)
		if success {
			SetConfigValueTo(Data.File, "index", fmt.Sprint(Data.Count))
			success = SetConfigValueTo(Data.File, "title", Data.Title)
			SetConfigValueTo(Data.File, "amiuser", Data.AMIUser)
			SetConfigValueTo(Data.File, "amipass", Data.AMIPass)
		} else {
			Data.Message = "Unable to write configuration"
			Data.MessageType = "errormessage"
		}
	}
	return
}

type ModifyFileType struct {
	ResponseType
	Result string `json:"result"`
}

func SaveRemoteFile(url string, file string, content string) (res ModifyFileType, CallErr error) {
	var obj = map[string]string{}
	obj["filename"] = file
	obj["content"] = content
	bytes, err := json.Marshal(obj)
	if err != nil {
		WriteLog("Error in SaveRemoteFile Marshal data: " + err.Error())
	}

	bytes, CallErr = restCallURL(url+"ModifyFile", bytes)
	if CallErr != nil {
		WriteLog("Error in SaveRemoteFile restCallURL  " + url + ": " + CallErr.Error())
	} else {
		err = json.Unmarshal(bytes, &res)
		if err != nil {
			WriteLog("Error in SaveRemoteFile Unmarshal response: " + err.Error())
		}
	}
	return
}

func UpdateUserPassword(id int, newPass string) (success bool, message string) {
	err := UpdatePassword(id, GetMD5(newPass))
	if err != nil {
		WriteLog("Error in UpdateUser: " + err.Error())
		message = err.Error()
	}
	success = err == nil
	return
}

type AMIResult struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage"`
	Result       string `json:"Result"`
}

func callAMICommand(pbxfile string, command string) (ResponseType, error) {

	command = "action:command\ncommand:" + command
	return callAMI(pbxfile, command)
}

func callAMI(pbxfile string, command string) (response ResponseType, Callerr error) {

	url := GetConfigValueFrom(pbxfile, "url", "")
	var obj = map[string]string{}
	username := GetConfigValueFrom(pbxfile, "amiuser", "")
	secret := GetConfigValueFrom(pbxfile, "amipass", "")
	obj["username"] = username
	obj["secret"] = secret
	obj["command"] = command
	bytes, err := json.Marshal(obj)
	if err != nil {
		WriteLog("Error in callAMI Marshal: " + err.Error())
	} else {
		if url != "" {
			if string(url[len(url)-1]) != "/" {
				url += "/"
			}
		}
		bytes, Callerr = restCallURL(url+"CallAMI", bytes)
		if Callerr != nil {
			WriteLog("Error in callAMI restCallURL: " + Callerr.Error())
		} else {
			err = json.Unmarshal(bytes, &response)
			if err != nil {
				WriteLog("Error in callAMI Unmarshal: " + err.Error())
			}
		}
	}
	return
}

type FileDataType struct {
	FileName   string
	Files      []ListFileType
	LastUpdate string
	Content    string
	Include    []IncludeType
	FileList   bool
}

func GetFileData(fileName string, pbxfile string) (File FileDataType, CallErr error) {
	File.FileName = fileName
	if fileName != "" {
		var obj = map[string]string{}
		obj["filename"] = fileName
		bytes, err := json.Marshal(obj)
		if err != nil {
			WriteLog("Error in GetFileData Marshal: " + err.Error())
		}

		url := GetConfigValueFrom(pbxfile, "url", "")
		if url != "" {
			if string(url[len(url)-1]) != "/" {
				url += "/"
			}
		}

		if fileName == "all" {
			File.FileName = "All Files"
			File.FileList = true
			File.Files, CallErr = GetFilesList(url)
		} else {
			File, CallErr = GetFileContents(url, fileName, bytes)
		}
	}
	return
}

type IncludeType struct {
	Line   string
	Action string
}

func GetIncludedFiles(content string, action string) (include []IncludeType) {

	arr := strings.Split(content, "\n")

	for _, line := range arr {
		if strings.Contains(line, "#include") && (!strings.Contains(line, ";") || strings.Index(line, ";") > strings.Index(line, "#include")) {
			includeIndex := strings.Index(line, "#include") + len("#include")
			line = line[includeIndex:len(line)]
			line = strings.TrimSpace(line)
			include = append(include, IncludeType{Action: action, Line: line})
		}
	}
	return
}

type FileType struct {
	ResponseType
	Content  string `json:"content"`
	Filetime string `json:"filetime"`
}

func GetFileContents(url, fileName string, bytes []byte) (Data FileDataType, CallErr error) {
	Data.FileName = fileName
	bytes, CallErr = restCallURL(url+"GetFile", bytes)
	if CallErr != nil {
		WriteLog("Error in GetFileContents restCallURL: " + CallErr.Error())
	} else {
		var response FileType
		err := json.Unmarshal(bytes, &response)
		if err != nil {
			WriteLog("Error in GetFileContents Unmarshal: " + err.Error())
		}
		if response.Success {
			// Display last updated time
			if response.Filetime != "" {
				FileTimeDot := strings.Contains(response.Filetime, ".")
				if FileTimeDot || strings.Contains(response.Filetime, "+") {
					terminateAt := "."
					if !FileTimeDot {
						terminateAt = "+"
					}
					response.Filetime = response.Filetime[0:strings.Index(response.Filetime, terminateAt)]
				}
				Data.LastUpdate = response.Filetime
			}

			Data.Include = GetIncludedFiles(response.Content, "Files?file=")
			Data.Content = response.Content
		} else {
			CallErr = errors.New(response.Message)
		}
	}
	return
}

type ListFilesType struct {
	ResponseType
	Files []string `json:"files"`
}

type ListFileType struct {
	Name  string
	NewTR bool
}

func GetFilesList(url string) (Files []ListFileType, CallErr error) {
	var bytes []byte
	bytes, CallErr = restCallURL(url+"ListFiles", nil)
	if CallErr != nil {
		WriteLog("Error in GetFilesList RestCallURL: " + CallErr.Error())
	} else {
		var response ListFilesType
		err := json.Unmarshal(bytes, &response)
		if err != nil {
			WriteLog("Error in GetFilesList Unmarshal: " + err.Error())
		} else if response.Success {
			var record ListFileType
			for i, file := range response.Files {
				i1 := i + 1
				record.NewTR = i1%6 == 0
				record.Name = file
				Files = append(Files, record)
			}
		}
	}
	return
}

func GetBackupFilesList(url string, bytes []byte, fileName string) (Files []string, Error error) {
	bytes, Error = restCallURL(url+"ListFiles", bytes)
	if Error != nil {
		WriteLog("Error in GetBackupFilesList restCallURL: " + Error.Error())
	} else {
		var response ListFilesType
		err := json.Unmarshal(bytes, &response)
		if err != nil {
			WriteLog("Error in GetBackupFilesList: " + err.Error())
		} else if response.Success {

			for _, file := range response.Files {
				originalFileName := file[0 : strings.Index(file, "conf")+4]
				if fileName == originalFileName {
					Files = append(Files, file)
				}
			}
		} else {
			Error = errors.New(response.Message)
		}
	}
	return
}

func doRetrieve(r *http.Request, fileName string, url string) {
	if r.FormValue("retrieve") != "" {
		// saveobj = JSONObject();
		// saveobj.put("filename", fileName);
		// saveobj.put("content", request.getParameter("content"));
		// String requestText = saveobj.toJSONString();
		// String resultText = General.restCallURL(url + "ModifyFile", requestText);
		// JSONParser saveparser = new JSONParser();
		// JSONObject saveresObj = (JSONObject) saveparser.parse(resultText);
		// boolean res = (Boolean.valueOf(saveresObj.get("success").toString()));
		// if (res) {
		//     out.println("<p class=infomessage>File Replaced</p>");
		//     out.println("<a href='Files?file=" + fileName + "'>View (Read only)</a>");
		//     Web.displayReloadLink(fileName, out);
		// }
		// else {
		//     out.println("<p class=errormessage>Error: " + saveresObj.get("message").toString() + "</p>");
		// }

	}
}

type BackupFileContentType struct {
	FileTime string
	Content  string
}

func GetBackupFileContents(url string, bytes []byte, originalFileName string, backupFileName string) (Data BackupFileContentType, Error error) {
	bytes, Error = restCallURL(url+"GetFile", bytes)
	var Response GetFileResponseType
	err := json.Unmarshal(bytes, &Response)
	if err != nil {
		WriteLog("Error in GetBackupFileContents Unmarshal: " + err.Error())
	} else {
		if Response.Success {
			if strings.Contains(Response.Filetime, ".") || strings.Contains(Response.Filetime, "+") {
				terminateAt := "."
				if !strings.Contains(Response.Filetime, ".") {
					terminateAt = "+"
				}
				Data.FileTime = Response.Filetime[0:strings.Index(Response.Filetime, terminateAt)]
			}
			Data.Content = Response.Content
		} else {
			Error = errors.New(Response.Message)
		}
	}
	return
}
