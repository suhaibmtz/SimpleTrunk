package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	Count := 0
	for _, file := range InfoFiles {
		record := GetPBXFile(file.Name())
		if record.IsStc {
			Count += 1
		}
		if (Count+1)%2 == 0 {
			record.Color = "#dAbaa7"
		} else {
			record.Color = "#dececa"
		}
		record.NewTR = Count%5 == 0
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

func GetFile(url, filename string) (response GetFileResponseType, err error) {
	obj := make(map[string]string)
	obj["filename"] = filename
	data, _ := json.Marshal(obj)
	if url != "" {
		if string(url[len(url)-1]) != "/" {
			url += "/"
		}
	}
	var bytes []byte
	bytes, err = restCallURL(url+"GetFile", data)
	if err == nil {
		err = json.Unmarshal(bytes, &response)
		if err == nil {
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
				}

			} else {
				err = errors.New(response.Message)
			}
		}
	}
	return
}

func GetRemoteFile(url string) (text string, err error) {

	var Response GetFileResponseType
	Response, err = GetFile(url, "/etc/simpletrunk/stagent.ini")
	if err != nil {
		WriteLog("Error in GetRemoteFile restCallURL: " + err.Error())
	} else {
		if Response.Success {
			text = Response.Content
		} else if err == nil {
			text = "Error: " + Response.Message
		}
	}
	return
}

func SavePbx(Data *PBXType, edit bool) (success bool) {
	CheckFolder()
	dir := GetPBXDir()
	if !strings.Contains(Data.File, ".") {
		Data.File += ".stc"
	}

	file := dir + Data.File
	if FileExist(Data.File) && !edit {
		success = false
		Data.Message = "Already Exist"
		Data.MessageType = "errormessage"
	} else {
		Data.Url = strings.ReplaceAll(Data.Url, `\`, "")
		success = SetConfigValueTo(file, "url", Data.Url)
		if success {
			SetConfigValueTo(file, "index", fmt.Sprint(Data.Count))
			success = SetConfigValueTo(file, "title", Data.Title)
			SetConfigValueTo(file, "amiuser", Data.AMIUser)
			SetConfigValueTo(file, "amipass", Data.AMIPass)
		} else {
			Data.Message = "Unable to write configuration"
			Data.MessageType = "errormessage"
		}
	}
	return
}

func SaveRemoteFile(url string, file string, content string) (res ResponseType, CallErr error) {
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
	Files      []TableListType
	LastUpdate string
	Content    string
	Include    []IncludeType
	FileList   bool
}

func GetFileData(fileName string, pbxfile string) (File FileDataType, CallErr error) {
	File.FileName = fileName
	if fileName != "" {

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
			File, CallErr = GetFileContents(url, fileName)
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

func GetFileContents(url, fileName string) (Data FileDataType, CallErr error) {
	Data.FileName = fileName
	var response GetFileResponseType
	response, CallErr = GetFile(url, fileName)
	if CallErr != nil {
		WriteLog("Error in GetFileContents restCallURL: " + CallErr.Error())
	} else {
		Data.LastUpdate = response.Filetime
		Data.Include = GetIncludedFiles(response.Content, "Files?file=")
		Data.Content = response.Content
	}
	return
}

type ListFilesType struct {
	ResponseType
	Files []string `json:"files"`
}

type TableListType struct {
	Name  string
	NewTR bool
}

func GetFilesList(url string) (Files []TableListType, CallErr error) {
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
			var record TableListType
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

func doRetrieve(r *http.Request, fileName string, url string) (err error, Retrieve bool) {
	if r.FormValue("retrieve") != "" {
		Retrieve = true
		var Response ResponseType
		Response, err = SaveRemoteFile(url, fileName, r.FormValue("content"))
		if err == nil && !Response.Success {
			err = errors.New(Response.Message)
		}
	}
	return
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

func SplitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		return strings.ContainsRune(seps, r)
	}
	return strings.FieldsFunc(s, splitter)
}

func Atoi(str string) (i int) {
	var err error
	i, err = strconv.Atoi(str)
	if err != nil {
		i = -1
	}
	return
}

func extractLineNumbers(token string) (dp DiffPosition) {

	if strings.Contains(token, "a") {
		dp.Type = "a"
	} else if strings.Contains(token, "d") {
		dp.Type = "d"
	} else if strings.Contains(token, "c") {
		dp.Type = "c"
	}

	linesNumber := SplitAny(token, "adc")
	if len(linesNumber) > 1 {
		firstFileLines := strings.Split(linesNumber[0], ",")
		dp.FirstFileStartPos = Atoi(firstFileLines[0])
		if len(firstFileLines) >= 2 {
			dp.FirstFileEndPos = Atoi(firstFileLines[1])

		} else {
			dp.FirstFileEndPos = Atoi(firstFileLines[0])
		}

		SecondFileLines := strings.Split(linesNumber[1], ",")
		dp.SecondFileStartPos = Atoi(SecondFileLines[0])
		if len(SecondFileLines) >= 2 {
			dp.SecondFileEndPos = Atoi(SecondFileLines[1])

		} else {
			dp.SecondFileEndPos = Atoi(SecondFileLines[0])
		}
	}

	return dp
}

func diff(res ResponseType) (dpArr []DiffPosition) {
	token := strings.Split(res.Result, "\n")

	count := 0
	diffToken := ""
	for i := 0; i < len(token); i++ {
		diffToken = token[i]
		if len(diffToken) > 0 {
			if diffToken[0] != '>' && diffToken[0] != '<' && diffToken[0] != '-' {
				dpArr = append(dpArr, extractLineNumbers(diffToken))
			}
		}

		count++
	}
	return
}

func FileToCompare(dpArr []DiffPosition, ContentArr []string, isBackup bool) (file []LineType) {
	Count := 0
	for i := 0; i < len(ContentArr); i++ {
		var Line LineType
		if i >= len(ContentArr) {
			Line.LineN = i + 1
			Line.Line = "\t"
		} else {
			if len(dpArr) == 0 {
				Line.LineN = i + 1
				Line.Line = ContentArr[i]
			} else {
				if dpArr[Count].SecondFileEndPos <= 0 && Count < len(dpArr)-1 {
					Count++
				}
				startPoint := dpArr[Count].SecondFileStartPos - 1
				endPoint := dpArr[Count].SecondFileEndPos
				if startPoint < 0 {
					startPoint = len(ContentArr)
				}
				if endPoint < 0 {
					endPoint = len(ContentArr)
				}
				if startPoint == i {
					for startPoint < endPoint && i < len(ContentArr) {
						var Line LineType
						Line.Line = ContentArr[i]
						Line.LineN = i + 1
						if isBackup {
							switch dpArr[Count].Type {
							case "a":
								Line.SpanColor = "#02a322"
								Line.Span = "▼"
								break
							case "d":
								Line.Color = "#FFA0B4"
								break
							case "c":
								Line.Color = "#A0C8FF"
								break
							}
						} else {
							switch dpArr[Count].Type {
							case "a":
								Line.Color = "#B4FFB4"
								break
							case "d":
								Line.Span = "▼"
								Line.SpanColor = "#ff3658"
								break
							case "c":
								Line.Color = "#A0C8FF"
								break
							}
						}
						startPoint++
						if startPoint < endPoint {
							i++
						}
						file = append(file, Line)

					}
					if Count < len(dpArr)-1 {
						Count++
					}

				} else {
					Line.LineN = i + 1
					Line.Line = ContentArr[i]
				}

			}

		}
		if Line.LineN != 0 {
			file = append(file, Line)
		}
	}
	return
}

func CompareFile(Org, Backup FileDataType, originalFileName, backupFileName string, dpArr []DiffPosition) (OrgLines []LineType, BackUpLines []LineType) {

	originalContent := Org.Content
	backupContent := Backup.Content

	originalContentArr := strings.Split(originalContent, "\n")
	backupContentArr := strings.Split(backupContent, "\n")
	OrgLines = FileToCompare(dpArr, originalContentArr, false)
	BackUpLines = FileToCompare(dpArr, backupContentArr, true)
	return
}

func GetNodes(Content string) (nodes []string) {
	arr := strings.Split(Content, "\n")
	for _, line := range arr {
		line = strings.TrimSpace(line)
		if strings.Index(line, "[") == 0 && strings.Index(line, "]") > 2 {
			line = line[1:strings.Index(line, "]")]
			nodes = append(nodes, line)
		}
	}

	return nodes
}

func addNewNode(fileName, nodename, content, url string) (message string) {

	saveobj := make(map[string]string)
	saveobj["filename"] = fileName
	saveobj["nodename"] = nodename
	saveobj["content"] = content
	bytes, _ := json.Marshal(saveobj)
	bytes, err := restCallURL(url+"AddNode", bytes)
	if err != nil {
		WriteLog("Error in addNewNode: " + err.Error())
		message = err.Error()
	} else {
		var Response ResponseType
		err = json.Unmarshal(bytes, &Response)
		if err != nil {
			WriteLog("Error in addNewNode Unmarshal: " + err.Error())
			message = err.Error()
		} else {
			if !Response.Success {
				message = Response.Message
			}
		}
	}
	return
}

func GetNodeContent(fileName, AgentUrl, nodename string) (content, message string) {

	res, err := GetFile(AgentUrl, fileName)
	if err != nil {
		message = err.Error()
	} else {
		if res.Success {
			arr := strings.SplitAfter(res.Content, "\n")

			started := false
			found := false
			for _, line := range arr {
				line = strings.TrimSpace(line)
				if !found && strings.Contains(line, nodename) {
					started = true
					found = true
				} else if started && strings.Contains(line, "[") && (strings.Index(line, "[") < 5) {
					started = false
				}

				if started {
					if line != "" {
						content += line + "\n"
					}
				}

			}
		} else {
			message = res.Message
		}
	}
	return
}

func SaveNode(r *http.Request, fileName, nodename, url string) (res ResponseType, err error) {
	if r.FormValue("save") != "" {
		saveobj := make(map[string]string)
		saveobj["filename"] = fileName
		saveobj["nodename"] = nodename
		saveobj["content"] = r.FormValue("content")
		bytes, _ := json.Marshal(saveobj)
		bytes, err = restCallURL(url+"ModifyNode", bytes)
		if err != nil {
			WriteLog("Error in SaveNode: " + err.Error())
		} else {
			//displayReloadLink(fileName, out)
			err = json.Unmarshal(bytes, &res)
		}

	}
	return
}

func GetReloadCommand(fileName string) (command, caption string) {

	if fileName == "extensions.conf" || fileName == "sip.conf" {
		command = "dialplanreload"
		caption = "Reload Dialplan"
		if fileName == "sip.conf" {
			command = "sipreload"
			caption = "Reload SIP"
		}
	}
	return
}

func callCLI(url, command string) (res ResponseType, err error) {

	obj := make(map[string]string)
	obj["command"] = command
	data, _ := json.Marshal(obj)

	bytes, err := restCallURL(url+"Command", data)
	if err == nil {
		err = json.Unmarshal(bytes, &res)
		if err != nil {
			WriteLog("Error in CallCLI Unmarshal: " + err.Error())
		}
	} else {
		WriteLog("Error in CallCLI: " + err.Error())
	}
	return
}

func executeShell(command, url string) (res ResponseType, err error) {

	obj := make(map[string]string)
	obj["command"] = command
	data, _ := json.Marshal(obj)

	var bytes []byte
	bytes, err = restCallURL(url+"Shell", data)
	if err != nil {
		WriteLog("Error in executeShell Callurl: " + err.Error())
	} else {
		err = json.Unmarshal(bytes, &res)
	}
	return
}

func GetLogTail(url, file string, lines string) (res GetFileResponseType, err error) {
	obj := make(map[string]any)
	obj["file"] = file
	obj["lines"] = lines
	data, _ := json.Marshal(obj)

	var bytes []byte
	bytes, err = restCallURL(url+"GetLogTail", data)
	if err != nil {
		WriteLog("Error in getLogTail CallURL: " + err.Error())
	} else {
		json.Unmarshal(bytes, &res)
	}
	return
}

type Operation struct {
	Success   bool
	ErrorCode int
	Message   string
	Size      int64
}

func DownloadFile2(fileURL string, urlParameters []byte, contentType string, w http.ResponseWriter) (op Operation, err error) {
	/*

			req,_ := http.NewRequest("GET",fileURL,bytes.NewReader(urlParameters))
			req.Header.Set("Content-Type", contentType)
		   // req.setDoOutput(true)
		   // writer.write(urlParameters)
		   WriteLog("URL " + fileURL)
		   WriteLog("Parameters: " + urlParameters)
		   responseCode = httpConn.getResponseCode()

		   result := ""
		   // always check HTTP response code first
		   if (responseCode == HttpURLConnection.HTTP_OK) {

		       // opens input stream from the HTTP connection
			req.inp
		       //String saveFilePath = filePath

		       // opens an output stream to save into file
		      // outputStream = new FileOutputStream(saveFilePath)

		       long size = 0
		       int bytesRead
		       byte[] buffer = new byte[1024]

		       while ((bytesRead = inputStream.read(buffer)) != -1) {
		           outputStream.write(buffer, 0, bytesRead)
		           size = size + bytesRead
		       }
		       if (size < 2048) {
		           byte[] text = new byte[(int)size]
		           System.arraycopy(buffer, 0, text, 0, (int)size)
		           String str = new String(text)
		           result = result + str

		       }


		       outputStream.close()
		       inputStream.close()

		       op.success = size > 2048
		       op.size = size

		       if (!op.success) {
		           try {
		             JSONParser parser = new JSONParser()
		             JSONObject obj = (JSONObject)parser.parse(result)
		             op.message = obj.get("message").toString()
		           }
		           catch (Exception ex){
		               op.success = false
		               op.errorCode = 5
		               op.message = "Error while parsing result: " + ex.toString()
		               General.writeEvent("Error : " + ex.toString())
		           }
		       }
		   } else {
		       op.success = false
		       op.errorCode = 5
		       op.message = "HTTP download Error"
		       General.writeEvent("HTTP error: " + responseCode)
		   }
		   httpConn.disconnect()
	*/
	return
}
func DownloadFile(fileURL string, urlParameters []byte, contentType string, outputStream io.Writer) (*Operation, error) {
	op := &Operation{}

	// Open URL and handle error
	u, err := url.Parse(fileURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(urlParameters))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)

	// Create client for making the request
	client := &http.Client{}

	// Send request and handle response
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP download error: %d", resp.StatusCode)
	}

	// Extract filename from URL or headers if possible
	filename := u.Path
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		parts := strings.Split(contentDisposition, ";")
		for _, part := range parts {
			if strings.TrimSpace(part) == "filename=" {
				filename = strings.Trim(strings.SplitN(part, "=", 2)[1], "\"")
				break
			}
		}
	}

	// Calculate estimated download speed
	contentLength := resp.ContentLength
	startTime := time.Now()

	// Copy data and calculate downloaded bytes
	written := 0
	buffer := make([]byte, 1024)
	for {
		n, err := io.ReadFull(resp.Body, buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		outputStream.Write(buffer[:n])
		written += n
	}

	// Calculate and display download speed
	elapsed := time.Since(startTime)
	speed := float64(written) / elapsed.Seconds() / 1024.0
	fmt.Printf("\nDownloaded %s in %.2fs at %.2f MB/s\n", filename, elapsed.Seconds(), speed)

	// Check file size and parse JSON if small
	op.Success = true
	op.Size = int64(written)
	if contentLength < 2048 {
		op.Message = string(buffer[:written])
	} else {
		// Implement JSON parsing and error handling for your specific needs
		// ...
	}

	return op, nil
}
