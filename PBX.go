package main

import (
	"encoding/json"
	"errors"
	"net/http"
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

func GetPBXHeader(username, page, selected string, r *http.Request) (Head HeaderType) {
	Head = GetHeader(username, "PBX", r)
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
		Header := GetPBXHeader(User.Name, "PBX", "", r)
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
			Data.HeaderType = GetPBXHeader(User.Name, page, "", r)
			Data.FileName = r.FormValue("file")
			if Data.FileName == "" {
				Data.FileName = "sip.conf"
			}
			var err error
			Data.Pre, err = doAddNode(r, AgentUrl, &Data)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			Data.Nodes, err = GetExtensions(AgentUrl, Data.FileName, r, Data.Title, Data.IsExten, page)
			if err != nil {
				Data.ErrorMessage(err.Error())
			}
			Data.DisplayAdd = r.FormValue("add") != ""
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
			Data.HeaderType = GetPBXHeader(User.Name, "Dialplans", "", r)
			var err error
			action := r.FormValue("action")
			Data.DisplayAdd = action == "displayadd"
			var nodename string
			Data.Pre, nodename, err = addNewContext(r, AgentUrl)
			if err != nil {
				Data.ErrorMessage(err.Error())
			} else if nodename != "" {
				Data.InfoMessage("New node " + nodename + " has been added")
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

func GetCallInfo(pbxfile, channel string) (callInfo CallInfoType) {

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
						call := GetCallInfo(pbxfile, channelID)
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

				record.CallInfo = GetCallInfo(pbxfile, channel)
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
			Data.HeaderType = GetPBXHeader(User.Name, "Queues", "", r)
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
	Function string
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
			switch function {
			case "calls":
				selected = "Active Channels"
			case "cdr":
				selected = "Last CDRs"
			}
			Data.HeaderType = GetPBXHeader(User.Name, "Monitor", selected, r)
			Data.Function = function
			err := mytemplate.ExecuteTemplate(w, "Monitor.html", Data)
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
