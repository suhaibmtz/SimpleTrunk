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

type CompareLineType struct {
	Original string
	BackUp   string
}

type CompareFilesType struct {
	HeaderType
	Lines    []CompareLineType
	Original string
	BackUp   string
}

func CompareFiles(w http.ResponseWriter, r *http.Request) {
	exist, User := CheckSession(r)
	if exist {

		pbx := GetCookieValue(r, "file")
		pbxfile := GetPBXDir() + pbx
		if FileExist(pbxfile) {
			Data := GetAdvancedHeader(User.Name, "Comapre", "", r)
			r.ParseForm()
			originalFileName := r.FormValue("originalfilename")
			backupFileName := r.FormValue("backupfilename")
			if r.FormValue("CompareFiles") != "" {

				if originalFileName != "" && backupFileName != "" {
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
					bytes, _ = json.Marshal(obj)
					bytes, err = restCallURL(url+"Shell", bytes)
					if err != nil {
						Data.Message = "Error: " + err.Error()
						Data.MessageType = "errormessage"
					} else {
						displayCompareFile(Org, Back, originalFileName, backupFileName)

					}
				}
			}
			err := mytemplate.ExecuteTemplate(w, "advanced.html", Data)
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
func displayCompareFile(final PrintWriter out, JSONObject firstResObj,JSONObject secondResObj,  String originalFileName , String backupFileName , ArrayList<DiffPosition> dpArr) {
        
        String originalContent = "";
        if (Boolean.valueOf(firstResObj.get("success").toString())) {
            originalContent = firstResObj.get("content").toString();
        }
        String backupContent = "";
        if (Boolean.valueOf(secondResObj.get("success").toString())) {
            backupContent = secondResObj.get("content").toString();
        }
        
        
        String[] originalContentArr = originalContent.split("\\r?\\n" , -1);
        String[] backupContentArr = backupContent.split("\\r?\\n" , -1);
        
        for (int i = 0 ; i < backupContentArr.length ; i++ ){
            //out.println("<p>"+ i+"  "+backupContentArr[i]+"</p>");
        }
        
        //int contentLength = (originalContentArr.length > backupContentArr.length)? originalContentArr.length:backupContentArr.length;
            
        out.println("<br><br><br>");
        out.println("<div style=' margin:auto; overflow:hidden;' >");
        
            out.println("<table width='50%' style='float: left; display: block;' >");
                out.println("<tbody>");
                    
                    out.println("<tr> <th> </th>");
                    out.println(" <th> <h3>"+originalFileName+"</h3></th> </tr>");
                    
                        int originCount = 0 ;              
                        for (int i = 0;i<originalContentArr.length; i++ ){
                           
                            if(i  >= originalContentArr.length){
                                out.println("<tr>");
                                out.println("<td>"+(i+1) +"</td>");
                                out.println("<td>  \t </td>");
                                out.println("</tr>");
                            }else{
                                if (dpArr.size() == 0 ){
                                    
                                    out.println("<tr>");
                                    out.println("<td>"+(i+1) +"</td>");
                                    out.println("<td>"+originalContentArr[i] +" </td>");
                                    out.println("</tr>");                                   
                                }else{
                                    if (dpArr.get(originCount).secondFileEndPos  <= 0){
                                        originCount++ ;
                                    } 
                                    int startPoint = dpArr.get(originCount).secondFileStartPos -1  ;
                                    int endPoint = dpArr.get(originCount).secondFileEndPos    ;
                                    if (startPoint == i ){
                                        while (startPoint < endPoint ){
                                            out.println("<tr>");
                                            out.println("<td>"+(i+1) +"</td>");
                                            switch(dpArr.get(originCount).type){
                                                case 'a':
                                                    out.println("<td bgcolor='#B4FFB4'>"+originalContentArr[i] +" </td>");
                                                    break ;
                                                case 'd':
                                                    //out.println("<td bgcolor='#FFA0B4'>"+originalContentArr[i] +" </td>");
                                                    out.println("<td>"+originalContentArr[i] +"<span style='color:#ff3658 ;'> ▼</span>  </td>");
                                                    break ;
                                                case 'c':
                                                    out.println("<td bgcolor='#A0C8FF'>"+originalContentArr[i] +" </td>");
                                                    break ;                                                
                                           }

                                            out.println("</tr>");
                                            startPoint++ ;
                                           if (startPoint < endPoint){
                                               i++ ;
                                           }

                                        }
                                        if (originCount < dpArr.size()-1 ){
                                             originCount++ ;
                                        }

                                   }else{
                                       out.println("<tr>");
                                       out.println("<td>"+(i+1) +"</td>");
                                       out.println("<td>"+originalContentArr[i] +" </td>");
                                       out.println("</tr>");
                                   }                                    
                                    
                                    
                                }

                               
                             }
                        }   
                
                
                out.println("</tbody>");
                out.println("</table>");       
                
                        
                            /////////////////////////////////////////////////////

                            
                            
            out.println("<table width='50%'; style='float: left; display: block';>");
                out.println("<tbody>");
                    out.println("<tr> <th>  </th> ");
                    out.println(" <th> <h3>"+backupFileName+"</h3></th> </tr>");
                        int backupCount = 0 ;                             
                        for (int i = 0;i< backupContentArr.length ; i++ ){  
                            
                            if(i  >= backupContentArr.length){
                                out.println("<tr>");
                                out.println("<td>"+(i+1) +"</td>");
                                out.println("<td>  \t </td>");
                                out.println("</tr>");
                            }else{
                                if (dpArr.size() == 0 ){
                                    
                                    out.println("<tr>");
                                    out.println("<td>"+(i+1) +"</td>");
                                    out.println("<td>"+backupContentArr[i] +" </td>");
                                    out.println("</tr>");                                   
                                }else{
                                   if (dpArr.get(backupCount).firstFileEndPos  <= 0){
                                        backupCount++ ;
                                    } 
                                    int startpoint = dpArr.get(backupCount).firstFileStartPos -1  ;
                                    int endPoint = dpArr.get(backupCount).firstFileEndPos   ;
                                    if (startpoint== i ){
                                        while (startpoint < endPoint){
                                           out.println("<tr>");
                                           out.println("<td>"+(i+1) +"</td>");
                                           switch(dpArr.get(backupCount).type){
                                                case 'a':
                                                    //out.println("<td bgcolor='#B4FFB4'>"+backupContentArr[i] +" </td>");
                                                    out.println("<td>"+backupContentArr[i] +"<span style='color:#02a322 ;'> ▼</span>  </td>");
                                                    break ;
                                                case 'd':
                                                    out.println("<td bgcolor='#FFA0B4'>"+backupContentArr[i] +" </td>");
                                                    break ;
                                                case 'c':
                                                    out.println("<td bgcolor='#A0C8FF'>"+backupContentArr[i] +" </td>");
                                                    break ;

                                           }

                                           out.println("</tr>");
                                           startpoint++ ;
                                           if (startpoint < endPoint){
                                               i++ ;
                                           }
                                           
                                        }
                                       if (backupCount < dpArr.size()-1){
                                           backupCount++ ;
                                       }


                                    }else{
                                       out.println("<tr>");
                                       out.println("<td>"+(i+1) +"</td>");
                                       out.println("<td>"+backupContentArr[i] +" </td>");
                                       out.println("</tr>");
                                    }                        
                                }
                            }                                                          
                        }
                out.println("</tbody>");
            out.println("</table>");  
        out.println("</div> <br>");    
    }