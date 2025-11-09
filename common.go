package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/motaz/codeutils"
)

func WriteLog(event string) {
	fmt.Println(event)
	err := codeutils.WriteToLog(event, simpletrunkPath+"/log/simpletrunk")
	if err != nil {
		fmt.Println("WriteLog Error: " + err.Error())
	}
}

func GetConfigValueFrom(file, param, def string) (value string) {

	CheckFolder()
	value = codeutils.GetConfigValue(file, param)

	if value == "" {
		value = def
	}
	return
}

func SetConfigValueTo(file, param, value string) (success bool) {

	CheckFolder()
	success = codeutils.SetConfigValue(file, param, value)
	return
}

func GetConfigValue(param, def string) (value string) {
	return GetConfigValueFrom(simpletrunkPath+"/simpletrunk.ini", param, def)
}

func GetMD5(text string) string {
	return codeutils.GetMD5(text)
}

type ResponseType struct {
	Success   bool   `json:"success"`
	Errorcode int    `json:"errorcode"`
	Message   string `json:"message"`
	Result    string `json:"result"`
}

type GetFileResponseType struct {
	ResponseType
	Content  string `json:"content"`
	Filetime string `json:"filetime"`
}

func RemoveColons() {

	Files := GetPBXFilesInfo()
	dir := GetPBXDir()
	for _, f := range Files {
		old := GetPBXFileString(f.Name())
		fileStr := strings.ReplaceAll(old, `\`, "")
		if old != fileStr {
			file, err := os.OpenFile(dir+f.Name(), os.O_RDWR, 0)
			if err == nil {
				WriteLog("removed colons from " + f.Name())
				file.WriteString(fileStr)
			}
		}
	}
}

func restCallURL(url string, data []byte) (value []byte, err error) {

	var res *http.Response
	url = strings.TrimSpace(url)
	if data == nil {
		res, err = http.Get(url)
	} else {
		res, err = http.Post(url, "POST", bytes.NewReader(data))
	}
	if err == nil {
		value, err = io.ReadAll(res.Body)
	} else if strings.Contains(err.Error(), "URL cannot contain colon") {
		err = nil
		go RemoveColons()
		url = strings.ReplaceAll(url, `\`, "")
		value, err = restCallURL(url, data)
	}
	return
}

func GetCookieValue(r *http.Request, cookie string) (Value string) {

	coo, err := r.Cookie(cookie)
	if err == nil {
		Value = coo.Value
	}
	return
}
