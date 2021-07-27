package main

import(
		"fmt"
		"net"
		"bufio"
		"time"
		"strings"
)

const(
	address = "irc.libera.chat:6667"
	channel = "#base48-t"
	nick	= "osw-bot2"
)

func main() {
	for{
		con, _ := net.Dial("tcp", address)
		cb := bufio.NewReader(con)

		ch := make(chan string)
		go checksw(con, ch)

		con.Write([]byte(fmt.Sprintf("NICK %s\n", nick)))
		con.Write([]byte(fmt.Sprintf("USER %s 0 * :improved open switch bot\n", nick)))
		con.Write([]byte(fmt.Sprintf("JOIN %s\n", channel)))

		for{
			str, err := cb.ReadString('\n')
			if err != nil { break }
			eval(str, con, ch)
		}
		time.Sleep(30 * time.Second)
	}
}

func eval(line string, con net.Conn, ch chan string){
	fmt.Println(line)
	str := strings.Fields(line)

	if str[0] == "PING"{
		fmt.Printf("Ping je tu: %s\n", str[1][1:])
		con.Write([]byte(fmt.Sprintf("PONG :%s\n", str[1][1:])))
	}
	if str[1] == "TOPIC" || str[1] == "332"{
		ch <- strings.Join(str[3:], " ")[1:]
	}
	if str[1] == "PRIVMSG" && str[3] == ":.beacon" && len(str) >= 5{
		if str[4] == "on"{
			fmt.Println("BEACON ON")
		} else if str[4] == "off" {
			fmt.Println("BEACON OFF")
		}
	}
	if str[1] == "PRIVMSG" && str[3] == ":.info"{
		con.Write([]byte(fmt.Sprintf("PRIVMSG %s :%s\n", channel, "chod do pice")))
	}
}

func checksw(con net.Conn, ch chan string){
	var topic string
	for{
		select {
			case topic = <-ch:
			default:
		}

		fmt.Printf("TOPIC: %s\n", topic)
		time.Sleep(2 * time.Second)
	}
}

