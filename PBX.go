package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var pbxPages = []TabType{
	{Name: "Extensions", Value: "Extensions"},
	{Value: "Extensions?type=trunk", Name: "Trunks"},
	{Value: "Dialplans", Name: "Dialplans"},
	{Value: "Functions", Name: "Queues"},
	{Value: "Monitor", Name: "Monitor"},
}

func GetPBXHeader(User UserType, page, selected string, r *http.Request) (Head HeaderType) {
	Head = GetHeader(User, "PBX", r)
	Head.Tabs = append(Head.Tabs, TabsType{Tabs: pbxPages, Selected: page})
	var Tabs TabsType
	switch page {
	case "Monitor":
		Tabs.Selected = selected
		Tabs.Text = "Monitor"
		Tabs.Tabs = []TabType{
			{Name: "System", Value: "?function=system"},
			{Name: "Active Channels", Value: "?function=calls"},
			{Name: "Last CDRs", Value: "?function=cdr"},
		}
	}
	if Tabs.Tabs != nil {
		Head.Tabs = append(Head.Tabs, Tabs)
	}
	return
}

func PBX(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		Header := GetPBXHeader(User, "PBX", "", r)
		err := mytemplate.ExecuteTemplate(w, "pbxpage.html", Header)
		if err != nil {
			WriteLog("Error in PBX execute template: " + err.Error())
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type ExtensionNodeType struct {
	NodeInfoType
	Username string
	Host     string
	Context  string
}

type ExtensionsType struct {
	HeaderType
	IsExten    bool
	FileName   string
	Nodes      []ExtensionNodeType
	DisplayAdd bool
	Type       string
	Title      string
	Pre        string
	File       string
}

func GetExtensionNode(node NodeInfoType) (Extension ExtensionNodeType) {
	Extension.NodeInfoType = node
	Extension.Username = node.GetProperty("username")
	Extension.Host = node.GetProperty("host")
	Extension.Context = node.GetProperty("context")
	return
}

func GetExtensions(Aurl, fileName string, r *http.Request, title string, isExten bool, Type string) (Extensions []ExtensionNodeType, err error) {

	var res GetFileResponseType
	res, err = GetFile(Aurl, fileName)
	if err == nil {
		if res.Success {
			nodes := getNodesWithInfo(res.Content)

			reverse := GetCookieValue(r, "reverse") == "yes"
			GetIncludedFiles(res.Content, "Extensions?type="+Type+"&file=")
			if reverse {
				for i := len(nodes) - 1; i >= 0; i-- {
					node := nodes[i]
					if node.IsExtension() == isExten {
						Extensions = append(Extensions, GetExtensionNode(node))
					}
				}
			} else {
				for _, node := range nodes {
					if node.IsExtension() == isExten {
						Extensions = append(Extensions, GetExtensionNode(node))
					}
				}
			}
		} else {
			err = errors.New(res.Message)
		}
	}
	return
}

func doAddNode(r *http.Request, Aurl string, Data *ExtensionsType) (pre string, err error) {
	if r.FormValue("addnode") != "" {
		saveobj := make(map[string]string)
		fileName := r.FormValue("file")
		if fileName == "" {
			fileName = "sip.conf"
		}
		saveobj["filename"] = fileName
		nodeName := "[" + r.FormValue("nodename") + "]"
		saveobj["nodename"] = nodeName
		content := "username=" + r.FormValue("username") + "\n"
		content += "type=" + r.FormValue("siptype") + "\n"
		content += "host=" + r.FormValue("host") + "\n"
		if r.FormValue("context") != "" {
			content += "context=" + r.FormValue("context") + "\n"
		}
		if r.FormValue("secret") != "" {
			content += "secret=" + r.FormValue("secret") + "\n"
		}
		if r.FormValue("additional") != "" {
			content = content + r.FormValue("additional")
		}
		if r.FormValue("preview") != "" { // Preview only
			pre = nodeName + "\n" + content
		} else // Create actual SIP node
		{
			saveobj["content"] = content
			data, _ := json.Marshal(saveobj)
			var bytes []byte
			bytes, err = restCallURL(Aurl+"AddNode", data)
			if err == nil {
				var res ResponseType
				json.Unmarshal(bytes, &res)
				if res.Success {
					Data.InfoMessage("New node " + nodeName + " has been added")
				} else {
					err = errors.New(res.Message)
				}
			}
		}
	}
	return
}

func Extensions(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxname := GetCookieValue(r, "file")
		pbx := GetPBXDir() + pbxname
		if FileExist(pbx) && pbxname != "" {
			AgentUrl := GetConfigValueFrom(pbx, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var Data ExtensionsType
			page := "Extensions"
			Data.Title = "Extension"
			Data.Type = r.FormValue("type")
			Data.IsExten = true
			if Data.Type == "trunk" {
				page = "Trunks"
				Data.Title = "Trunk"
				Data.IsExten = false
			} else {
				Data.Type = "ext"
			}
			Data.HeaderType = GetPBXHeader(User, page, "", r)
			Data.FileName = r.FormValue("file")
			if Data.FileName == "" {
				Data.FileName = "sip.conf"
			}
			var err error
			if User.Admin {
				Data.Pre, err = doAddNode(r, AgentUrl, &Data)
				if err != nil {
					Data.ErrorMessage(err.Error())
				}
			}
			Data.Nodes, err = GetExtensions(AgentUrl, Data.FileName, r, Data.Title, Data.IsExten, page)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			Data.DisplayAdd = r.FormValue("add") != "" && User.Admin
			err = mytemplate.ExecuteTemplate(w, "Extensions.html", Data)
			if err != nil {
				WriteLog("Error in Extensions execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type DialplansType struct {
	HeaderType
	DisplayAdd bool
	Nodes      []TableListType
	Pre        string
}

func Dialplans(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxname := GetCookieValue(r, "file")
		pbx := GetPBXDir() + pbxname
		if FileExist(pbx) && pbxname != "" {
			AgentUrl := GetConfigValueFrom(pbx, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var Data DialplansType
			Data.HeaderType = GetPBXHeader(User, "Dialplans", "", r)
			var err error
			action := r.FormValue("action")
			Data.DisplayAdd = action == "displayadd" && User.Admin
			if User.Admin {
				var nodename string
				Data.Pre, nodename, err = addNewContext(r, AgentUrl)
				if err != nil {
					Data.ErrorMessage(err.Error())
				} else if nodename != "" {
					Data.InfoMessage("New node " + nodename + " has been added")
				}
			}
			Data.Nodes, err = GetDialplans(AgentUrl)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			err = mytemplate.ExecuteTemplate(w, "Dialplans.html", Data)
			if err != nil {
				WriteLog("Error in Extensions execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

func addNewContext(r *http.Request, Aurl string) (pre string, nodeName string, err error) {

	if r.FormValue("addcontext") != "" {
		contextname := "[" + r.FormValue("contextname") + "]"
		digits := "_X."

		aDisgits := r.FormValue("digits")

		if aDisgits == "any" {
			digits = "_X."
		} else if aDisgits == "fixed" {
			digits = r.FormValue("fixedvalue")
		} else if aDisgits == "pattern" {
			digits = r.FormValue("paternvalue")
		}

		// Plan
		content := "exten => " + digits + ",1,NoOp()\n"
		answer := r.FormValue("answer")
		if answer != "" && answer == "1" {
			content += "  same => n,answer()\n"
		}

		play := r.FormValue("play")
		if play != "" && play == "1" {
			content += "  same => n,playback(" + r.FormValue("recording") + ")\n"
		}

		dial := r.FormValue("dial")
		if dial != "" && dial == "1" {
			content += "  same => n,dial(" + r.FormValue("dialto") + ")\n"
		}
		content += "  same => n,hangup()\n"
		if r.FormValue("preview") != "" { // Preview only
			pre = contextname + "\n" + content
		} else {
			message := addNewNode("extensions.conf", contextname, content, Aurl)
			if message != "" {
				err = errors.New(message)
			} else {
				nodeName = contextname
			}
		}
	}
	return
}

type FunctionsType struct {
	HeaderType
	Now     string
	Funcs   TabsType
	IsBusy  bool
	Keyword string
	Queues  []QueueType
	Count   int
	Waiting []QueueType
	WCount  int
}

type CallInfoType struct {
	CallerID    string
	Application string
	Time        string
}

type QueueType struct {
	Queue       string
	DisplayLine bool
	Member      string
	CallInfo    CallInfoType
	Channel     string
	Line        string
	Eq          bool
}

func GetChannelID(pbxfile, queue, agent string) (channelIDs []string) {

	op, _ := callAMI(pbxfile, "core show channels concise")
	if op.Success {
		lines := strings.Split(op.Message, "\n")
		for _, line := range lines {
			if strings.Contains(line, queue) && strings.Contains(line, agent) && !strings.Contains(line, ";1") {
				channelIDs = append(channelIDs, line[0:strings.Index(line, "!")])
			}

		}
	}
	return

}

func FuncsGetCallInfo(pbxfile, channel string) (callInfo CallInfoType) {

	callInfo.CallerID = ""
	op, _ := callAMI(pbxfile, "core show channel "+channel)
	if op.Success {
		lines := strings.Split(op.Message, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Caller ID Name:") {
				callInfo.CallerID = strings.TrimSpace(line[strings.Index(line, ":")+1 : len(line)])
			}
			if strings.Contains(line, "Application") {
				callInfo.Application = strings.TrimSpace(line[strings.Index(line, ":")+1 : len(line)])
			}
			if strings.Contains(line, "Elapsed") {
				callInfo.Time = strings.TrimSpace(line[strings.Index(line, ":")+1 : len(line)])
			}

			if strings.Contains(line, "Connected Line ID:") && strings.Contains(callInfo.CallerID, "N/A") {
				callInfo.CallerID = strings.TrimSpace(line[strings.Index(line, ":")+1 : len(line)])
			}

			if strings.Contains(line, "DNID Digits:") && strings.Contains(callInfo.CallerID, ("N/A")) {
				callInfo.CallerID = strings.TrimSpace(line[strings.Index(line, ":")+1 : len(line)])
			}
		}
	}
	return callInfo

}

func GetStatusOf(pbxfile string, has bool, keyword string) (isBusy bool, queues []QueueType, count int, newKeyword string, err error) {

	isBusy = keyword == "Busy"

	var result ResponseType
	result, err = callAMICommand(pbxfile, "queue show")
	if result.Success {
		lines := strings.Split(result.Message, "\n")
		count = 0
		var queue string
		for _, line := range lines {
			var record QueueType
			if strings.Contains(line, "holdtime") {
				queue = line[0:strings.Index(strings.TrimSpace(line), " ")]
			}
			if isBusy {
				record.DisplayLine = (strings.Contains(line, "Agent/") || strings.Contains(line, "SIP/") || strings.Contains(line, "Local/")) &&
					isBusy && (strings.Contains(line, "in call")) && (!strings.Contains(line, "Not in use"))
			} else if keyword == "paused" {
				record.DisplayLine = (strings.Contains(line, "Agent/") || strings.Contains(line, "SIP/") || strings.Contains(line, "Local/")) &&
					(has && strings.Contains(line, keyword) || (!has && !strings.Contains(line, keyword)))

			}

			if record.DisplayLine {
				count++

				// Agent/Member
				// SIP/1010 (ringinuse disabled)[1;36;40m (dynamic)[0m[0m[0m[0m ([1;31;40mUnavailable[0m) has taken no calls yet
				//(Local/875@agents from Agent:875) (ringinuse disabled)[1;36;40m (dynamic)[0m[0m[0m[0m ([1;32;40mInvalid[0m) has taken no calls yet
				if strings.Contains(line, "/") && strings.Contains(line, "(") {
					if strings.Index(line, "(") < strings.Index(line, "/") {
						line = line[strings.Index(line, "(")+1 : len(line)]

					}

					record.Member = strings.TrimSpace(line[strings.Index(line, "/")+1 : strings.Index(line, "(")])
					line = line[strings.Index(line, "/")+1 : len(line)]
					if strings.Contains(record.Member, "@") {
						record.Member = record.Member[0:strings.Index(record.Member, "@")]
					}
					// Remove Member additional string
					//(Local/875@agents from Agent:875) (ringinuse disabled)
					if strings.Index(line, ")") < strings.Index(line, "(") {
						line = line[strings.Index(line, ")")+1 : len(line)]
					}
				}

				if queue == "" {
					queue = "-"
				}

				if strings.Contains(line, "(") {
					line = line[strings.Index(line, "("):len(line)]
				}

				// Option
				//out.println("<td  style='font-size:12'>" + line.substring(0, line.indexOf(")") +1 ) + "</td>");

				if strings.Contains(line, ")") {
					line = line[strings.Index(line, ")")+1 : len(line)]
				}
				record.CallInfo.CallerID = "-"
				record.CallInfo.Time = "-"
				channelIDs := GetChannelID(pbxfile, record.Queue, record.Member)
				if len(channelIDs) != 0 {
					for _, channelID := range channelIDs {
						call := FuncsGetCallInfo(pbxfile, channelID)
						if len(call.CallerID) > len(record.CallInfo.CallerID) {
							record.CallInfo = call
						}
					}
				}

				// Info
				record.Line = line
				/*if (! isBusy){
					out.println("<td><form method=post>");
					out.println("<input type=hidden name=member value='" + member + "' />");
					if (has && keyword.equals("paused")) {
					    out.println("<input type=submit name=unpause value='Unpause' />");
					}
					else if (! has && keyword.equals("paused")) {
					    out.println("<input type=submit name=pause value='Pause' />");
					}
					out.println("</form></td>");
				    }
				*/
			}
			if queue != "-" && queue != "" && record != (QueueType{}) {
				record.Queue = queue
				queue = ""
				queues = append(queues, record)
			}
		}

		if count == 0 {
			if !has {
				keyword = "Un" + keyword
			}
		}
	}
	newKeyword = keyword
	return
}

func GetWaiting(pbxfile string) (count int, queues []QueueType, err error) {

	op, _ := callAMI(pbxfile, "queue show")
	if op.Success {
		text := op.Message
		queue := ""

		lines := strings.Split(text, "\n")
		started := false
		lastQueue := ""
		for _, line := range lines {
			var record QueueType

			if strings.Contains(line, "holdtime") && strings.Contains(line, " ") {
				queue = strings.TrimSpace(line[0:strings.Index(line, " ")])
			}
			if strings.TrimSpace(line) == "" {
				started = false
			}
			if started {

				channel := ""
				if strings.Contains(line, ".") && strings.Contains(line, "(") {
					channel = strings.TrimSpace(line[strings.Index(line, ".")+1 : strings.Index(line, "(")])
				}

				//String info[] = General.getCallInfo(pbxfile, callid);
				//if ((info != null) && (info.length > 30)){
				//String callerID = getFieldValue("Caller ID", info);
				//String application = getFieldValue("Data:", info);

				record.Eq = lastQueue != queue

				lastQueue = queue

				line = line[strings.Index(line, "("):len(line)]
				record.Line = line
				record.Channel = channel

				record.CallInfo = FuncsGetCallInfo(pbxfile, channel)
				count++

			}
			//}
			if strings.Contains(line, "Callers:") {

				started = true
			}

			if queue != "" && record != (QueueType{}) {
				record.Queue = queue
				queues = append(queues, record)
			}
		}

	}
	return
}

func Functions(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxname := GetCookieValue(r, "file")
		pbx := GetPBXDir() + pbxname
		if FileExist(pbx) && pbxname != "" {
			AgentUrl := GetConfigValueFrom(pbx, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var Data FunctionsType
			Data.HeaderType = GetPBXHeader(User, "Queues", "", r)
			var err error
			function := r.FormValue("function")
			if function == "" {
				function = "talk"
			}
			Data.Funcs.Selected = function
			Data.Funcs.Tabs = []TabType{
				{Name: "Queue-Active", Value: "active"},
				{Name: "Queue-Paused", Value: "paused"},
				{Name: "Talking/Waiting", Value: "talk"},
			}
			Data.Now = time.Now().Format("Mon Jan 2 15:04:05 MST 2006")
			var has bool
			var keyword string
			switch function {
			case "paused":
				keyword = "paused"
				has = true
			case "active":
				keyword = "paused"
				has = false
			case "talk":
				keyword = "Busy"
				has = true
				Data.WCount, Data.Waiting, _ = GetWaiting(pbx)
			}
			Data.IsBusy, Data.Queues, Data.Count, Data.Keyword, err = GetStatusOf(pbx, has, keyword)
			if err != nil {
				Data.ErrorMessage("Error: " + err.Error())
			}
			err = mytemplate.ExecuteTemplate(w, "Functions.html", Data)
			if err != nil {
				WriteLog("Error in Extensions execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}

type MonitorType struct {
	HeaderType
	Function  string
	Now       string
	CDRResult CDRResultType
	Calls     ActiveChannelsType
	SystemStatusType
	CallsCount int
}

type CDRResultType struct {
	Header []string   `json:"header"`
	Data   [][]string `json:"data"`
}

type CDRResponseType struct {
	ResponseType
	Result CDRResultType `json:"result"`
}

func GetCDR(url string) (result CDRResultType, err error) {

	data, err := restCallURL(url+"GetLastCDR", nil)

	if err == nil {
		var res CDRResponseType
		json.Unmarshal(data, &res)

		if res.Success {
			result = res.Result
		}
	}
	return
}

type CallType struct {
	CallerID    string
	ID          string
	Extension   string
	Duration    string
	Application string
}

type ActiveChannelsType struct {
	Count int
	Calls []CallType
}

type UsageLineType struct {
	IsFont bool
	Color  string
	Line   string
}

type SystemStatusType struct {
	Percent   string
	BGColor   string
	Time      string
	ProcCount int
	IP        string
	TopProc   []string
	Memory    string
	Lines     []UsageLineType
}

func GetSystemStatus(url string) (System SystemStatusType, err error) {

	// CPU Utilization
	var res ResponseType
	res, err = executeShell("top -b  | head -14 ", url)
	if err == nil {
		loadStr := res.Result
		toplines := strings.Split(loadStr, "\n")
		percent := "-1"
		if len(toplines) > 2 {
			percent = toplines[2]
		}
		//%Cpu(s): 31.3 us, 9.0 sy, 0.0 ni, 59.7 id, 0.0 wa, 0.0 hi, 0.0 si, 0.0 st %
		percent = percent[0:strings.Index(percent, "id")]
		percent = percent[strings.LastIndex(percent, ",")+1 : len(percent)]
		percent = strings.TrimSpace(percent)
		utilization, _ := strconv.ParseFloat(percent, 32)
		utilization = 100 - utilization
		result, _ := executeShell("nproc", url)
		var bgcolor string
		proc := strings.ReplaceAll(result.Result, "\n", "")
		procCount, _ := strconv.Atoi(proc)

		bgcolor = "#AAFFAA"
		if utilization > 100 {
			bgcolor = "#990000"
		} else if utilization > 90 {
			bgcolor = "#FF5555"
		} else if utilization > 50 {
			bgcolor = "#FFFFaa"
		}
		if utilization > 100 {
			utilization = 100
		} else if utilization == 0 {
			bgcolor = "#FFFFFF"
		}

		System.BGColor = bgcolor
		System.Percent = fmt.Sprintf("%0.1f", utilization)

		result, _ = executeShell("date", url)
		System.Time = result.Result
		System.ProcCount = procCount

		result, _ = executeShell("ip a", url)
		ipList := strings.Split(result.Result, "\n")
		for _, ip := range ipList {
			if strings.Contains(ip, "inet ") && !strings.Contains(ip, "127.0.") {
				ip = strings.TrimSpace(ip)
				ip = ip[strings.Index(ip, " "):len(ip)]
				ip = strings.TrimSpace(ip)
				ip = ip[0:strings.Index(ip, " ")]
				System.IP = ip
				break
			}
		}
		for i := 6; i < len(toplines); i++ {
			System.TopProc = append(System.TopProc, toplines[i])
		}

		result, _ = executeShell("free -m", url)
		System.Memory = result.Result

		result, _ = executeShell("df -h", url)
		lines := strings.Split(result.Result, "\n")
		for _, line := range lines {
			var record UsageLineType
			record.Line = line
			if strings.Contains(line, "/") && strings.Contains(line, "%") {
				usageStr := line[strings.Index(line, " "):strings.Index(line, "%")]
				usageStr = strings.TrimSpace(usageStr)
				for strings.Contains(usageStr, " ") {
					usageStr = usageStr[strings.Index(usageStr, " "):len(usageStr)]
					usageStr = strings.TrimSpace(usageStr)
				}
				usage, _ := strconv.ParseFloat(usageStr, 32)
				if usage > 80 {
					record.Color = "brown"
					record.IsFont = true
				} else if usage > 60 {
					record.Color = "#ee7766"
					record.IsFont = true
				}
			}
			System.Lines = append(System.Lines, record)
		}
	}
	return
}

func GetActiveChannels(pbxfile string) (Calls ActiveChannelsType, message string, err error) {

	var res ResponseType
	res, err = callAMICommand(pbxfile, "core show channels concise")
	text := res.Message
	if err == nil {
		if len(text) < 150 {
			if strings.Contains(text, "Privilege") {
				message = "No active channels"
			} else {
				message = text
			}
		}
		lines := strings.Split(text, "\n")
		count := 0
		for _, line := range lines {
			if strings.Contains(line, "!") {
				callid := line[0:strings.Index(line, "!")]
				callid = strings.ReplaceAll(callid, "Output: ", "")
				count++
				// Get details call info
				info, _ := GetCallInfo(pbxfile, callid)
				if len(info) > 30 {
					var record CallType
					record.CallerID = getFieldValue("Caller ID:", info)
					record.ID = getFieldValue("UniqueID:", info)
					record.Extension = getFieldValue("Connected Line ID:", info)
					record.Duration = getFieldValue("Elapsed Time:", info)
					record.Application = getFieldValue("Application:", info)
					Calls.Calls = append(Calls.Calls, record)
				}
			}
		}
		Calls.Count = count
	}
	return
}

func Monitor(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {
		pbxname := GetCookieValue(r, "file")
		pbx := GetPBXDir() + pbxname
		if FileExist(pbx) && pbxname != "" {
			AgentUrl := GetConfigValueFrom(pbx, "url", "")
			if AgentUrl != "" {
				if string(AgentUrl[len(AgentUrl)-1]) != "/" {
					AgentUrl += "/"
				}
			}
			var Data MonitorType
			selected := "System"
			function := r.FormValue("function")
			if function == "" {
				function = "system"
			}
			var err error
			switch function {
			case "calls":
				selected = "Active Channels"
				var message string
				Data.Calls, message, err = GetActiveChannels(pbx)
				if message != "" {
					Data.InfoMessage(message)
				}
			case "cdr":
				selected = "Last CDRs"
				Data.CDRResult, err = GetCDR(AgentUrl)
			case "system":
				Data.SystemStatusType, err = GetSystemStatus(AgentUrl)
			}
			Data.HeaderType = GetPBXHeader(User, "Monitor", selected, r)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			Data.Function = function
			Data.Now = time.Now().Format("Mon Jan 2 15:04:05 MST 2006")
			err = mytemplate.ExecuteTemplate(w, "Monitor.html", Data)
			if err != nil {
				WriteLog("Error in Extensions execute template: " + err.Error())
			}
		} else {
			http.Redirect(w, r, "Home?m=Select%20PBX", http.StatusTemporaryRedirect)
		}
	} else {
		http.Redirect(w, r, "login", http.StatusTemporaryRedirect)
	}
}
