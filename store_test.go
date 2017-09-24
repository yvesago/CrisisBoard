package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gopkg.in/olahol/melody.v1"
	"os"
	"testing"
)

func deleteFile(file string) {
	// delete file
	var err = os.Remove(file)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func TestBoardModel(t *testing.T) {
	testDbName := "_test.sql"
	defer deleteFile(testDbName)

	db := InitDb(testDbName)

	var s melody.Session
	s.Set("db", db)

	b := Board{Cmd: "reg", Obj: "obj", Npt: "11:12", Ev: "ev", Bil: "bil", Act: "act", Com: "com"}

	r, _ := Stored(&s, b)
	//fmt.Printf("res %+v\n", r)
	assert.Equal(t, int64(1), r.Id, "Insert Sucess")

	b.Id = 1
	b.Obj = "obj1"
	r, _ = Stored(&s, b)
	assert.Equal(t, b.Obj, r.Obj, "Update Sucess")
	assert.Equal(t, int64(1), r.Id, "Update Sucess")

	/* now in main.go

	    b.Cmd = "val"
	    b.Obj = "obj3"
		r, _ = Stored(&s, b)
	    assert.Equal(t, b.Obj, r.Obj, "Validate Sucess")
	    assert.Equal(t, int64(2), r.Id, "New raw. Validate Sucess")
	*/

	b.Id = 0
	b.Obj = "obj2"
	r, _ = Stored(&s, b)
	assert.Equal(t, b.Obj, r.Obj, "Insert Sucess")
	assert.Equal(t, int64(2), r.Id, "Insert Sucess")

	var c Board

	c = Board{Cmd: "read:1"}
	r, _ = Stored(&s, c)
	assert.Equal(t, "obj1" , r.Obj, "read 1 Sucess")


	c = Board{Cmd: "current"}
	r, _ = Stored(&s, c)
	assert.Equal(t, int64(2) , r.Id, "last stored is 2")
}
