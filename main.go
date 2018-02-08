package main

/*

./go-bindata -o myweb.go web/*

go test

go build  -ldflags "-s" -o crisisboard *.go


*/

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/olahol/melody.v1"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

var version string = "0.0.1"

//const letterBytes = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
const letterBytes = "abcdefghijkmnopqrstuvwxyz23456789" // simpliest password

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	pass := RandStringBytes(8)
	if os.Getenv("CRISIS_KEY") != "" {
		pass = os.Getenv("CRISIS_KEY")
	}

	servPtr := flag.String("s", "", "Serveur")
	usrPtr := flag.String("u", "crise", "Utilisateur")
	filePtr := flag.String("f", "crisisboard.sql", "Fichier base de données")
	portPtr := flag.String("p", "5001", "Port")
	debugPtr := flag.Bool("d", false, "Debug mode")
	flag.Parse()

	p := *portPtr
	user := *usrPtr
	file := *filePtr
	serv := *servPtr
	debug := *debugPtr

	if debug == false {
		gin.SetMode(gin.ReleaseMode)
	}

	// Config server
	db := InitDb(file)

	r := gin.New()

	r.Use(gin.Recovery())
	if debug == true {
		r.Use(gin.Logger())
	}
	m := melody.New()
	m.Config.MaxMessageSize = 4294967296 //2^32

	addrs, _ := net.InterfaceAddrs()

	if serv == "" {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				serv = ipnet.IP.String()
				if ipnet.IP.To4() != nil { // prefer shorter IPv4 if available
					break
				}
			}
		}
		serv = "http://" + serv + ":" + p + "/share/"
	}

	fmt.Println("#--------------------------------------------#")
	fmt.Println(" ")
	fmt.Println("    Usage =>  http://localhost:" + p + "/  <=")
	fmt.Println(" ")
	fmt.Println("  Partage :")
	fmt.Println("  =========")
	fmt.Println("  Server: " + serv)
	fmt.Println("    Pass: " + pass)
	fmt.Println(" ")
	fmt.Println("  version: " + version)
	fmt.Println("#--------------------------------------------#")

	// Add Assets
	data, err := Asset("web/index.html")
	if err != nil {
		// asset was not found.
		fmt.Println(err)
	}
	dcssd, _ := Asset("web/med/default.min.css")
	dcssme, _ := Asset("web/med/medium-editor.min.css")
	djsme, _ := Asset("web/med/medium-editor.min.js")

	// Manage share auth
	auth := r.Group("/", gin.BasicAuthForRealm(gin.Accounts{
		user: pass,
	}, "Utilisateur: "+user))

	// Gin router
	auth.GET("/share", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	r.GET("/", func(c *gin.Context) {
		if c.ClientIP() == "::1" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		}
	})

	r.GET("/board/med/default.min.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css; charset=utf-8", dcssd)
	})

	r.GET("/board/med/medium-editor.min.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css; charset=utf-8", dcssme)
	})

	r.GET("/board/med/medium-editor.min.js", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/javascript", djsme)
	})

	// Websocket router
	r.GET("/board/ws", func(c *gin.Context) {
		ml := make(map[string]interface{})
		ml["cip"] = c.ClientIP()
		ml["db"] = db
		m.HandleRequestWithKeys(c.Writer, c.Request, ml)
	})

	// Manage websocket messages
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if string(msg) == "share" {
			// display share acess
			var as []*melody.Session
			as = append(as, s)
			byteArray := []byte("share--" + serv + "--" + pass)
			m.BroadcastMultiple(byteArray, as)
		} else if string(msg) == "current" {
			// read last registred values
			var as []*melody.Session
			as = append(as, s)
			b := Board{Cmd: "current"}
			json.Unmarshal(msg, &b)
			a, _ := Stored(s, b)
			byteArray, _ := json.Marshal(a)
			m.BroadcastMultiple(byteArray, as)
		} else {
			var b Board
			e := json.Unmarshal(msg, &b)
			if e != nil {
				fmt.Println(e)
			}

			l, _ := s.Get("cip")
			b.Ip = l.(string) // need type assertion
			//fmt.Printf("b %+v\n", b)
			//fmt.Printf("b.npt %v\n", b.Npt)

			var a Board
			if b.Cmd != "val" { // for "reg",  "read:..."
				a, e = Stored(s, b)
				if e != nil {
					fmt.Printf("store: %s %v", b.Cmd, e)
				}
			}
			if b.Cmd == "val" { // TODO: manage in store
				b.Cmd = "reg"
				a, e = Stored(s, b)
				if e != nil {
					fmt.Printf("store: val %v", e)
				}
				b.Id = 0
				a, e = Stored(s, b)
			}

			// TODO: really want to send read history to all clients ?
			byteArray, _ := json.Marshal(a)
			m.Broadcast(byteArray)
		}
	})

	r.Run(":" + p)
}
