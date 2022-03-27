package main

import(
		"fmt"
		"net"
		"time"
		"bufio"
		"strings"
		"log/syslog"
		"net/http"
		"encoding/json"
		"github.com/stianeikeland/go-rpio"
		"github.com/julienschmidt/httprouter"
)

const(
	address = "irc.libera.chat:6667"
	channel = "#base48"
	nick	= "osw-bot2"
	port	= 10001		// port for rest api
)

type Sw struct{
	os rpio.Pin
	cs rpio.Pin
	ol rpio.Pin
	cl rpio.Pin
	be rpio.Pin
	log *syslog.Writer
	oc bool
	lastch time.Time
}

type Rest struct {
	Open bool `json:"open"`
	Last string	`json:"lastchange"`
}

var sw Sw

func main() {
	sw.log, _ = syslog.New(syslog.LOG_INFO|syslog.LOG_ERR, "osw-bot2")
	sw.log.Info("Start")

	rpio.Open()
	sw.os = rpio.Pin(17)
	sw.cs = rpio.Pin(4)
	sw.ol = rpio.Pin(22)
	sw.cl = rpio.Pin(23)
	sw.be = rpio.Pin(21)
	sw.ol.Output();	sw.cl.Output();	sw.be.Output()

	rs := httprouter.New()		// rest aspi
	rs.GET("/", rest)
	go http.ListenAndServe(fmt.Sprintf("%s:%d", "0.0.0.0", port), rs)

	for{
		con, _ := net.Dial("tcp", address)
		cb := bufio.NewReader(con)
		sw.log.Info("Connecting")

		con.Write([]byte(fmt.Sprintf("NICK %s\n", nick)))
		con.Write([]byte(fmt.Sprintf("USER %s 0 * :improved open switch bot\n", nick)))
		con.Write([]byte(fmt.Sprintf("JOIN %s\n", channel)))

		ch := make(chan string)
		go checksw(con, ch) // check sw thread

		for{
			str, err := cb.ReadString('\n')
			if err != nil { sw.log.Err(err.Error()); break }
			eval(str, con, ch)
		}
		time.Sleep(5 * time.Minute)
	}
	rpio.Close()
}

func rest(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var re Rest
	re.Open = sw.oc
	re.Last = fmt.Sprintf("%s", sw.lastch)

	rej, _ := json.Marshal(re)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", rej)
}

func eval(line string, con net.Conn, ch chan string){
	fmt.Println(line)
	str := strings.Fields(line)

	if str[0] == "PING"{
		con.Write([]byte(fmt.Sprintf("PONG :%s\n", str[1][1:])))
	}
	if str[1] == "TOPIC" || str[1] == "332"{
		last := strings.SplitN(line[1:], ":", 2)
		if len(last) >= 2 {
			ch <- last[1]
		}
	}
	if str[1] == "PRIVMSG" && str[3] == ":.beacon" && len(str) >= 5{
		sw.log.Info(line)
		if str[4] == "on"{
			sw.be.High()
		} else if str[4] == "off" {
			sw.be.Low()
		}
	}
	if str[1] == "PRIVMSG" && str[3] == ":.info"{
		con.Write([]byte(fmt.Sprintf("PRIVMSG %s : Beacon: %d\n", channel, sw.be.Read())))
	}
}

func checksw(con net.Conn, ch chan string){
	var topic string
	var to int
	for{
		select { case topic = <-ch: default: }
		os := sw.os.Read()
		cs := sw.cs.Read()

		if os == 1 && cs == 0 && ! strings.HasPrefix(topic, "base open") &&
			len(topic) != 0 && to == 0{
			two := strings.SplitN(topic, "|", 2); last := ""; to = 10
			if len(two) >= 2 { last = two[1] }
			fmt.Printf("os %n, cs %n, topic: %s", os, cs, topic)
			con.Write([]byte(fmt.Sprintf("TOPIC %s :base open \\o/ |%s\n", channel, last)))
			sw.log.Info("base open")
			sw.oc = true
			sw.lastch = time.Now()
		}
		if os == 0 && cs == 1 && ! strings.HasPrefix(topic, "base closed") &&
			len(topic) != 0 && to == 0{
			two := strings.SplitN(topic, "|", 2); last := ""; to = 10
			if len(two) >= 2 { last = two[1] }
			fmt.Printf("os %n, cs %n, topic: %s", os, cs, topic)
			con.Write([]byte(fmt.Sprintf("TOPIC %s :base closed :( |%s\n", channel, last)))
			sw.log.Info("base closed")
			sw.oc = false
			sw.lastch = time.Now()
		}

		if strings.HasPrefix(topic, "base open") { sw.ol.Write(os); sw.oc = true }
		if strings.HasPrefix(topic, "base closed") { sw.cl.Write(cs); sw.oc = false }

		if to != 0{ to-- }
		time.Sleep(1 * time.Second)
	}
}
