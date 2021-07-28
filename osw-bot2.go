package main

import(
		"fmt"
		"net"
		"time"
		"bufio"
		"strings"
		"github.com/stianeikeland/go-rpio"
)

const(
	address = "irc.libera.chat:6667"
	channel = "#base48"
	nick	= "osw-bot2"
)

type Sw struct{
	os rpio.Pin
	cs rpio.Pin
	ol rpio.Pin
	cl rpio.Pin
	be rpio.Pin
}

func main() {
	for{
		con, _ := net.Dial("tcp", address)
		cb := bufio.NewReader(con)
		rpio.Open()

		sw := &Sw {
			os : rpio.Pin(17),
			cs : rpio.Pin(4),
			ol : rpio.Pin(22),
			cl : rpio.Pin(23),
			be : rpio.Pin(27),
		}

		sw.ol.Output();	sw.cl.Output();	sw.be.Output()

		ch := make(chan string)
		go checksw(con, ch, sw)

		con.Write([]byte(fmt.Sprintf("NICK %s\n", nick)))
		con.Write([]byte(fmt.Sprintf("USER %s 0 * :improved open switch bot\n", nick)))
		con.Write([]byte(fmt.Sprintf("JOIN %s\n", channel)))

		for{
			str, err := cb.ReadString('\n')
			if err != nil { break }
			eval(str, con, ch, sw)
		}

		rpio.Close()
		time.Sleep(2 * time.Minute)
	}
}

func eval(line string, con net.Conn, ch chan string, sw *Sw){
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

func checksw(con net.Conn, ch chan string, sw *Sw){
	var topic string
	var to int
	for{
		select { case topic = <-ch: default: }
		os := sw.os.Read()
		cs := sw.cs.Read()

		if os == 1 && cs == 0 && ! strings.HasPrefix(topic, "base open") &&
			len(topic) != 0 && to == 0{
			two := strings.SplitN(topic, "|", 2); last := ""; to = 5
			if len(two) >= 2 { last = two[1] }
			con.Write([]byte(fmt.Sprintf("TOPIC %s :base open \\o/ |%s\n", channel, last)))
		}
		if os == 0 && cs == 1 && ! strings.HasPrefix(topic, "base closed") &&
			len(topic) != 0 && to == 0{
			two := strings.SplitN(topic, "|", 2); last := ""; to = 5
			if len(two) >= 2 { last = two[1] }
			con.Write([]byte(fmt.Sprintf("TOPIC %s :base closed :( |%s\n", channel, last)))
		}

		if strings.HasPrefix(topic, "base open") { sw.ol.Write(os) }
		if strings.HasPrefix(topic, "base closed") { sw.cl.Write(cs) }

		if to != 0{ to-- }
		time.Sleep(1 * time.Second)
	}
}
