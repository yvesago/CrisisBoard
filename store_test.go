package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"gopkg.in/olahol/melody.v1"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
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
	assert.Equal(t, int64(1), r.Id, "Insert Success")

	b.Id = 1
	b.Obj = "obj1"
	r, _ = Stored(&s, b)
	assert.Equal(t, b.Obj, r.Obj, "Update Success")
	assert.Equal(t, int64(1), r.Id, "Update Success")

	b.Id = 0
	b.Obj = "obj2"
	r, _ = Stored(&s, b)
	assert.Equal(t, b.Obj, r.Obj, "Insert Success")
	assert.Equal(t, int64(2), r.Id, "Insert Success")

	var c Board

	c = Board{Cmd: "read:1"}
	r, _ = Stored(&s, c)
	assert.Equal(t, "obj1", r.Obj, "read 1 Success")

	c = Board{Cmd: "current"}
	r, _ = Stored(&s, c)
	assert.Equal(t, int64(2), r.Id, "last stored is 2")
}

func contains(s []string, t string) bool {
	for _, v := range s {
		if v == t {
			return true
		}
	}
	return false
}

func TestAsset(t *testing.T) {
	defer func() {
		deleteFile("t/web/index.html")
		deleteFile("t/web/")
		deleteFile("t/")
	}()
	_, err := Asset("web/index.html")
	if err != nil {
		// asset was not found.
		fmt.Println(err)
	}
	d, _ := AssetInfo("web/index.html")
	assert.Equal(t, "web/index.html", d.Name(), "name of asset")
	a := AssetNames()
	liste := []string{
		"web/index.html",
		"web/med/default.min.css",
		"web/med/medium-editor.min.css",
		"web/med/medium-editor.min.js",
	}
	for _, la := range a {
		assert.Equal(t, true, contains(liste, la), "asset in list")
	}

	a, _ = AssetDir("web/med")
	liste = []string{"default.min.css", "medium-editor.min.css", "medium-editor.min.js"}
	for _, la := range a {
		assert.Equal(t, true, contains(liste, la), "asset in list")
	}
	RestoreAssets("t/", "web/index.html")
	_, e := os.Stat("t/web/index.html")
	assert.Equal(t, nil, e, "restored file exist")
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestServer(t *testing.T) {
	testFileName := "_test.sql"
	defer deleteFile(testFileName)

	localPort := "5001"
	randomPass := RandStringBytes(12)
	assert.Equal(t, "ynvpoqhs5g3x", randomPass, "test 12 char rand string")

	gin.SetMode(gin.TestMode)
	r := gin.New()

	assert.Equal(t, "http://exemple.com", localServer("http://exemple.com", localPort), "don't change fixed server url")
	urlContains := regexp.MustCompile(`:5001/share/$`).MatchString

	localServUrl := localServer("", localPort)
	assert.Equal(t, true, urlContains(localServUrl), "create local server url")

	banner(localPort, localServUrl, randomPass, version)

	server(r, "http://exemple.com", "crise", "qwerty", testFileName, true)

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

	liste := []string{
		"/board/med/default.min.css",
		"/board/med/medium-editor.min.css",
		"/board/med/medium-editor.min.js",
	}
	for _, la := range liste {
		req, _ := http.NewRequest("GET", la, nil)
		resp1 = httptest.NewRecorder()
		r.ServeHTTP(resp1, req)
		assert.Equal(t, 200, resp1.Code, "template success")
	}

	/**
	  test basic auth
	  **/
	req, err = http.NewRequest("GET", "/share", nil)
	req.Header.Add("Authorization", "Basic "+basicAuth("crise", "qwerty"))
	if err != nil {
		fmt.Println(err)
	}

	resp1 = httptest.NewRecorder()
	r.ServeHTTP(resp1, req)
	//fmt.Printf("%+v\n", resp1.Body)
	assert.Equal(t, 200, resp1.Code, "basic auth success")

	/**
	  test websocket
	  **/

	s := httptest.NewServer(r)
	defer s.Close()

	h := http.Header{"Authorization": {"Basic " + basicAuth("crise", "qwerty")}}
	d := websocket.Dialer{}
	c, resp, err := d.Dial("ws://"+s.Listener.Addr().String()+"/share/board/ws", h)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode, "ok switching connect")

	c, resp, err = d.Dial("ws://"+s.Listener.Addr().String()+"/board/ws", nil)
	/*if err != nil {
		t.Fatal(err)
	}*/
	assert.Equal(t, 401, resp.StatusCode, "access denied whithout auth")

	d = websocket.Dialer{}
	c, resp, err = d.Dial("ws://"+s.Listener.Addr().String()+"/board/ws", h)
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
	assert.Equal(t, int64(2), respbv.Id, "register with validation: new Id")

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
