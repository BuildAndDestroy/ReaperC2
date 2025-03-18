package adminconsole

import (
	"log"
	"net"
	"strings"
)

func StartAdminConsole() {
	ln, err := net.Listen("tcp", ":31337")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Admin console listening on 31337...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go handleAdminCommand(conn)
	}
}

func handleAdminCommand(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		command := strings.TrimSpace(string(buf[:n]))
		if command == "exit" {
			conn.Write([]byte("Closing connection...\n"))
			return
		} else if command == "status" {
			conn.Write([]byte("Server running...\n"))
		} else {
			conn.Write([]byte("Unknown command\n"))
		}
	}
}
