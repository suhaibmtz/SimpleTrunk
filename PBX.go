package main

import (
	"encoding/json"
	"errors"
	"net/http"
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
