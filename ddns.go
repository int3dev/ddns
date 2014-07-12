package main

import (
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pboehm/ddns/connection"
	"log"
	"net"
	"os"
	"strings"
)

func HandleErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func RunBackend() {
	conn := connection.OpenConnection()
	defer conn.Close()

	fmt.Printf("OK\tDDNS Go Backend\n")

	bio := bufio.NewReader(os.Stdin)

	for {
		line, _, err := bio.ReadLine()
		HandleErr(err)

		parts := strings.Split(string(line), "\t")
		if len(parts) == 6 {
			query_name := parts[1]
			fqdn_parts := strings.Split(query_name, ".")

			query_class := parts[2]
			query_id := parts[4]

			if len(fqdn_parts) > 0 {
				if conn.HostExist(fqdn_parts[0]) {
					host := conn.GetHost(fqdn_parts[0])

					record := "A"
					if !host.IsIPv4() {
						record = "AAAA"
					}

					fmt.Printf("DATA\t%s\t%s\t%s\t10\t%s\t%s\n",
						query_name, query_class, record, query_id, host.Ip)
				}
			}
		}

		fmt.Printf("END\n")
	}

}

func RunWebService() {
	conn := connection.OpenConnection()
	defer conn.Close()

	r := gin.Default()
	r.GET("/new/:hostname", func(c *gin.Context) {
		hostname := c.Params.ByName("hostname")

		if conn.HostExist(hostname) {
			c.String(403, "This hostname has already been registered.")
			return
		}

		host := &connection.Host{Hostname: hostname, Ip: "127.0.0.1"}
		host.GenerateAndSetToken()

		conn.SaveHost(host)

		c.String(200, fmt.Sprintf(
			"Go to /update/%s/%s for updating your IP address",
			host.Hostname, host.Token))
	})

	r.GET("/update/:hostname/:token", func(c *gin.Context) {
		hostname := c.Params.ByName("hostname")
		token := c.Params.ByName("token")

		if !conn.HostExist(hostname) {
			c.String(404,
				"This hostname has not been registered or is expired.")
			return
		}

		host := conn.GetHost(hostname)

		if host.Token != token {
			c.String(403,
				"You have supplied the wrong token to manipulate this host")
			return
		}

		ip, _, err := net.SplitHostPort(c.Req.RemoteAddr)
		if err != nil {
			c.String(500, "You sender IP address is not in the right format")
		}

		host.Ip = ip
		conn.SaveHost(host)

		c.String(200, fmt.Sprintf("Your current IP is %s", ip))
	})

	r.Run(":8080")
}

func main() {

	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]

	switch cmd {
	case "backend":
		log.Printf("Starting PDNS Backend\n")
		RunBackend()
	case "web":
		log.Printf("Starting Web Service\n")
		RunWebService()
	default:
		usage()
	}
}

func usage() {
	log.Fatal("Usage: ./ddns [backend|web]")
}
