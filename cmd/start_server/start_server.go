package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/pkg/server"
)

func main() {
	hostRef := flag.String("host", "", "Hostname of the server")
	portRef := flag.Int("port", -1, "port number")
	versionRef := flag.Bool("version", false, "display version")
	flag.Parse()

	// Parameter: version
	version := *versionRef
	if version {
		fmt.Println(lsp.GetServerVersion())
		return
	}

	// Command: stdio
	if len(os.Args) > 1 && os.Args[1] == "stdio" {
		lsp.StartServerStdio()
		return
	}

	// Parameter: host
	host := *hostRef
	if host == "" {
		host = os.Getenv("LSP_SERVER_HOST")

		if host == "" {
			host = "localhost"
		}
	}

	// Parameter: port
	port := *portRef
	if port == -1 {
		portEnv := os.Getenv("PORT")
		if portEnv == "" {
			fmt.Print("No port provided. Use --port or define PORT variable")
			return
		}

		var err error

		port, err = strconv.Atoi(portEnv)

		if err != nil {
			fmt.Printf(
				"The \"PORT\" environment variable is not a valid number (value: %s)",
				portEnv,
			)
			return
		}

		if port <= 0 || port > 65535 {
			fmt.Printf(
				"The \"PORT\" environment variable is not a valid port number (value: %d)",
				port,
			)
			return
		}
	}

	lsp.StartServer(port, host)
}
