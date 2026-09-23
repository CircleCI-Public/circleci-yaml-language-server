package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/logging"
	lsp "github.com/CircleCI-Public/circleci-yaml-language-server/internal/server"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/version"
)

func main() {
	hostRef := flag.String("host", "", "Hostname of the server")
	portRef := flag.Int("port", -1, "port number")
	schemaRef := flag.String("schema", "", "Location of the schema (optional, uses built-in schema if not provided)")
	versionRef := flag.Bool("version", false, "display version")
	stdioRef := flag.Bool("stdio", false, "Use stdio instead of socket to communicate")
	debugRef := flag.Bool("debug", true, "Log debug messages, including every request (-debug=false for info and above)")
	flag.Parse()

	// Parameter: version
	if *versionRef {
		fmt.Println(version.Server)
		return
	}

	logging.Setup(*debugRef)

	// Parameter: schema
	// If no schema is provided via flag or env, the embedded schema will be used.
	schema := *schemaRef
	if schema == "" {
		schema = os.Getenv("SCHEMA_LOCATION")
	}

	if schema != "" && !path.IsAbs(schema) {
		cwd, err := os.Getwd()

		if err != nil {
			slog.Error("resolving schema path", "schema", schema, "err", err)
			panic(err)
		}
		schema = path.Join(cwd, schema)
	}

	// Command: stdio
	if *stdioRef {
		lsp.StartServerStdio(schema)
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
	portEnv := os.Getenv("PORT")
	if port == -1 && portEnv != "" {
		var err error

		port, err = strconv.Atoi(portEnv)

		if err != nil {
			slog.Error("the PORT environment variable is not a valid number", "value", portEnv)
			return
		}

		if port <= 0 || port > 65535 {
			slog.Error("the PORT environment variable is not a valid port number", "value", port)
			return
		}
	} else {
		slog.Info("no port defined: the server will find a free port")
	}

	lsp.StartServer(port, host, schema)
}
