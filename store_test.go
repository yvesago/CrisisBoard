package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"gopkg.in/olahol/melody.v1"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func init() {
	rand.Seed(1) // fix rand fo tests
}

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
	assert.Equal(t, "obj1", r.Obj, "read 1 Sucess")

	c = Board{Cmd: "current"}
	r, _ = Stored(&s, c)
	assert.Equal(t, int64(2), r.Id, "last stored is 2")
}

func TestServer(t *testing.T) {
	testFileName := "_test.sql"
	defer deleteFile(testFileName)

	assert.Equal(t, "ynvpoqhs5g3x", RandStringBytes(12), "test 12 char rand string")

	gin.SetMode(gin.TestMode)
	r := gin.New()

	data, err := Asset("web/index.html")
	if err != nil {
		// asset was not found.
		fmt.Println(err)
	}

	/*
		// Manage share auth
		auth := r.Group("/", gin.BasicAuthForRealm(gin.Accounts{
			user: pass,
		}, "Utilisateur: "+user))

		// Gin router
		auth.GET("/share", func(c *gin.Context) {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	*/

	server(r, data, "http://exemple.com", "qwerty", testFileName, true)

	/**
	  test template
	  **/
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		fmt.Println(err)
	}

	resp1 := httptest.NewRecorder()
	r.ServeHTTP(resp1, req)
	//fmt.Printf("%+v\n", resp1.Body)
	assert.Equal(t, 200, resp1.Code, "template success")

	/**
	  test websocket
	  **/

	s := httptest.NewServer(r)
	defer s.Close()

	d := websocket.Dialer{}
	c, resp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/board/ws", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode, "ok switching connect")

	/**
	test share info
	**/
	o1 := []byte(`share`)
	err = c.WriteMessage(websocket.TextMessage, o1)
	if err != nil {
		t.Fatal(err)
	}

	_, sharews, _ := c.ReadMessage()
	//fmt.Printf("%+v\n", string(sharews))
	assert.Equal(t, "share--http://exemple.com--qwerty", string(sharews), "load contains some <b>text</b>")

	/**
	manage boards
	**/

	// register first board
	b := Board{Cmd: "reg", Obj: "obj", Npt: "11:12", Ev: "ev", Bil: "bil", Act: "act", Com: "com"}
	err = c.WriteJSON(b)
	if err != nil {
		t.Fatal(err)
	}

	var respb Board
	c.ReadJSON(&respb)

	assert.Equal(t, "reg", respb.Cmd, "test return cmd")
	assert.Equal(t, int64(1), respb.Id, "register but whithout validation")

	// validate first board and create a new board
	b = Board{Id: respb.Id, Cmd: "val", Obj: "obj", Npt: "11:12", Ev: "ev", Bil: "bil", Act: "act", Com: "com"}
	err = c.WriteJSON(b)
	if err != nil {
		t.Fatal(err)
	}

	var respbv Board
	c.ReadJSON(&respbv)

	assert.Equal(t, "reg", respbv.Cmd, "test return cmd")
	assert.Equal(t, int64(2), respbv.Id, "register whith validation: new Id")

	// register don't change id
	b = Board{Id: respbv.Id, Cmd: "reg", Obj: "obj", Npt: "11:12", Ev: "ev", Bil: "bil", Act: "act", Com: "com"}
	err = c.WriteJSON(b)
	if err != nil {
		t.Fatal(err)
	}

	var respbvr Board
	c.ReadJSON(&respbvr)

	assert.Equal(t, "reg", respbvr.Cmd, "test return cmd")
	assert.Equal(t, respbv.Id, respbvr.Id, "register : same Id")
	assert.NotEqual(t, respbv.Updated, respbvr.Updated, "register change updated")

	// read first board
	b = Board{Cmd: "read:1"}
	err = c.WriteJSON(b)
	if err != nil {
		t.Fatal(err)
	}

	var respbr Board
	c.ReadJSON(&respbr)

	assert.Equal(t, "read", respbr.Cmd, "test return cmd")
	assert.Equal(t, int64(1), respbr.Id, "read first validation")

	/**
	test load current board
	**/

	var respbc Board
	o := []byte(`current`)
	err = c.WriteMessage(websocket.TextMessage, o)
	if err != nil {
		t.Fatal(err)
	}

	c.ReadJSON(&respbc)
	//fmt.Printf("%+v\n", respb)

	assert.Equal(t, "11:12", respbc.Npt, "Next point board")
	assert.Equal(t, int64(2), respbc.Id, "register but whithout validation")

}
