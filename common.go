package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/motaz/codeutils"
)

func WriteLog(event string) {
	fmt.Println(event)
	codeutils.WriteToLog(event, simpletrunkPath+"/log/simpletrunk")
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
