package main

import (
	"database/sql"
	"io/fs"
	"io/ioutil"
	"os"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

var simpletrunkPath = ""
var DB *sql.DB

func init() {
	home, err := os.UserHomeDir()
	simpletrunkPath = home + "/simpletrunk"
	connectDB()
	if err != nil {
		WriteLog("Error in get home: " + err.Error())
	}
	CheckFolder()
}

func connectDB() {
	var err error
	DBPath := simpletrunkPath + "/simpletrunk.db"
	DB, err = sql.Open("sqlite3", DBPath)
	if err != nil {
		WriteLog("Error in Open DB at " + DBPath + " : " + err.Error())
	}
}

func GetPBXDir() string {
	dir := "pbxs/"
	if runtime.GOOS == "linux" {
		dir = simpletrunkPath + "/pbxs/"
	}
	if !FileExist(dir) {
		CheckFolder()
	}
	return dir
}

func GetPBXFilesInfo() (files []fs.FileInfo) {
	files, err := ioutil.ReadDir(GetPBXDir())
	if err != nil {
		WriteLog("Error in GetPBXFiles: " + err.Error())
	}
	return
}

func GetPBXFileString(name string) (file string) {
	bytes, err := os.ReadFile(GetPBXDir() + name)
	if err != nil {
		WriteLog("Error in GetPBXFileString: " + err.Error())
	} else {
		file = string(bytes)
	}
	return
}

func GetUser(what string, value string) (User UserType, err error) {
	row := DB.QueryRow(`select id,name,password from users where ` + what + ` = '` + value + `';`)
	err = row.Err()
	if err == nil {
		err = row.Scan(&User.ID, &User.Name, &User.Password)
	}
	return
}

func GetUsers() (Users []UserType, err error) {
	rows, err := DB.Query("select id,name,password from users;")
	if err == nil {
		for rows.Next() {
			var record UserType
			err = rows.Scan(&record.ID, &record.Name, &record.Password)
			Users = append(Users, record)
		}
	}
	return
}

func CreateFolder(dir string) {
	err := os.Mkdir(dir, 0755)
	if err != nil {
		WriteLog("Error in CreateFolder: " + err.Error())
	}
}

func InsertUser(username, password string) (err error) {
	_, err = DB.Exec("insert into users (name,password) VALUES(?,?)", username, password)
	return
}

func FileExist(Path string) bool {
	_, err := os.Stat(Path)
	return !os.IsNotExist(err)
}

func InsertTable(name, table string) (err error) {
	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS ` + name + ` ` + table)
	return
}

func GetSession(key string) (id int, err error) {
	row := DB.QueryRow("select id from session where key = '" + key + "'")
	err = row.Err()
	if err == nil {
		err = row.Scan(&id)
	}
	return
}

func SetSession(key string, id int) (err error) {
	_, err = DB.Exec("insert into session (id,key) Values(?,?)", id, key)
	return
}

func UpdatePassword(id int, newpass string) (err error) {
	_, err = DB.Exec("UPDATE users SET password = ? WHERE id = ?;", newpass, id)
	return
}
