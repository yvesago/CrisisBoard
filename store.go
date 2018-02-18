package main

import (
	"database/sql"
	"errors"
	"gopkg.in/olahol/melody.v1"
	//"fmt"
	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/gorp.v1"
	"log"
	"strings"
	"time"
)

type Board struct {
	Id      int64     `db:"id" json:"id"`
	Cmd     string    `db:"-" json:"cmd"`
	Obj     string    `db:"obj" json:"obj"`
	Npt     string    `db:"npt" json:"npt"`
	Ev      string    `db:"ev" json:"ev"`
	Act     string    `db:"act" json:"act"`
	Bil     string    `db:"bil" json:"bil"`
	Com     string    `db:"com" json:"com"`
	Ip      string    `db:"ip" json:"ip"`
	Created time.Time `db:"created" json:"created"` // or int64
	Updated time.Time `db:"updated" json:"updated"`
}

// Hooks : PreInsert and PreUpdate

func (b *Board) PreInsert(s gorp.SqlExecutor) error {
	b.Created = time.Now() // or time.Now().UnixNano()
	b.Updated = b.Created
	return nil
}

func (b *Board) PreUpdate(s gorp.SqlExecutor) error {
	b.Updated = time.Now()
	return nil
}

func InitDb(dbName string) *gorp.DbMap {
	db, err := sql.Open("sqlite3", dbName)
	checkErr(err, "sql.Open failed")
	//dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	dbmap.AddTableWithName(Board{}, "Board").SetKeys(true, "Id")
	err = dbmap.CreateTablesIfNotExists()
	checkErr(err, "Create tables failed")

	return dbmap
}

func checkErr(err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}

func Stored(s *melody.Session, b Board) (Board, error) {
	dba, e := s.Get("db")
	db := dba.(*gorp.DbMap)
	if e == false {
		return b, errors.New("no db in session")
	}

	cmd := strings.Split(b.Cmd, ":")

	switch cmd[0] {
	case "current":
		var br Board
		err := db.SelectOne(&br, "SELECT * FROM board ORDER BY id DESC LIMIT 1")
		br.Cmd = "current"
		if err != nil {
			return br, err
		}
		return br, nil
	case "reg":
		if b.Id == 0 {
			err := db.Insert(&b)
			if err != nil {
				return b, err
			}
		} else {
			_, err := db.Update(&b)
			if err != nil {
				return b, err
			}
		}
		return b, nil
	case "read":
		var br Board
		// obj, err := db.Get(Board{}, cmd[1])
		// br := obj.(*Board)
		err := db.SelectOne(&br, "SELECT * FROM board WHERE id=? LIMIT 1", cmd[1])
		br.Cmd = "read"
		if err != nil {
			return br, err
		}
		return br, nil
	default:
		return b, errors.New("Unk cmd")

	}

	//return b, errors.New("Unk cmd")
}
