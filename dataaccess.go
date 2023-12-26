package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"runtime"
	"time"

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
	go RemoveOldSession()
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
	// if !FileExist(dir) {
	// 	CheckFolder()
	// }
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
	row := DB.QueryRow(`select id,name,password,admin from users where ` + what + ` = '` + value + `';`)
	err = row.Err()
	if err == nil {
		var IsAdmin sql.NullBool
		err = row.Scan(&User.ID, &User.Name, &User.Password, &IsAdmin)
		User.Admin = IsAdmin.Bool
	}
	return
}

func GetUsers() (Users []UserType, err error) {
	rows, err := DB.Query("select id,name,password,admin from users;")
	defer rows.Close()
	if err == nil {
		for rows.Next() {
			var record UserType
			var Admin sql.NullBool
			err = rows.Scan(&record.ID, &record.Name, &record.Password, &Admin)
			record.Admin = Admin.Bool
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

func InsertUser(username, password string, admin bool) (err error) {
	_, err = DB.Exec("insert into users (name,password,admin) VALUES(?,?,?)", username, password, admin)
	return
}

func FileExist(Path string) bool {
	_, err := os.Open(Path)
	return err == nil
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

type CloseType struct {
	id  int
	key string
}

var toDel []CloseType

func Delete() {
	for true {
		for _, User := range toDel {
			key := User.key
			id := User.id
			query := "DELETE FROM session WHERE key = '" + key + "';"
			fmt.Println(query)
			_, err := DB.Exec(query)
			if err != nil {
				WriteLog("Error in Delete old Session: " + err.Error())
			} else {
				User, _ := GetUserByID(id)
				WriteLog("Deleted Old Session For " + User.Name + " with id " + fmt.Sprint(id))
			}
		}
		time.Sleep(time.Minute * 5)
	}
}

func RemoveOldSession() {
	go Delete()
	for true {
		rows, err := DB.Query("select key,time,id from session;")
		if err == nil {
			for rows.Next() {
				var Time time.Time
				var key string
				var id int
				rows.Scan(&key, &Time, &id)
				if time.Now().Sub(Time) >= time.Hour*24*8 {
					toDel = append(toDel, CloseType{key: key, id: id})
				}
			}
		}
		time.Sleep(time.Minute * 1)
	}
}
