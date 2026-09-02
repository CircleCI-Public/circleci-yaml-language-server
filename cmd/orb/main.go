// Command orb resolves an orb reference and prints what the language server
// would see. It is a debugging aid, the replacement for the old cmd/gql
// scratch tool.
//
// Usage:
//
//	go run cmd/orb/main.go [orb-reference]
//
// The reference defaults to circleci/go@1.7.1. Environment:
//
//	CIRCLE_TOKEN    used when set; without one, public orbs still resolve
//	CIRCLECI_HOST   defaults to https://circleci.com
//	ORB_BACKEND     "graphql" forces the GraphQL fallback; anything else, or
//	                unset, probes the host and uses V3 where it answers
//
// Forcing the backend is how the GraphQL fallback gets checked against real
// infrastructure: circleci.com serves GraphQL as well as V3, so both paths can
// be compared without a CircleCI Server instance.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

func main() {
	ref := "circleci/go@1.7.1"
	if len(os.Args) > 1 {
		ref = os.Args[1]
	}

	host := utils.CIRCLE_CI_APP_HOST_URL
	if fromEnv := os.Getenv("CIRCLECI_HOST"); fromEnv != "" {
		host = fromEnv
	}

	token := os.Getenv("CIRCLE_TOKEN")
	debug := os.Getenv("ORB_DEBUG") != ""

	var registry utils.OrbRegistry
	switch backend := os.Getenv("ORB_BACKEND"); backend {
	case "graphql":
		registry = utils.NewGraphQLOrbRegistry(host, token, "", debug)
	default:
		registry = utils.NewOrbRegistry(host, token, "", debug)
	}

	ctx := context.Background()

	orb, err := registry.FetchOrb(ctx, utils.OrbPackageName(ref))
	if err != nil {
		fmt.Fprintln(os.Stderr, "fetching orb:", err)
		os.Exit(1)
	}

	resolved, err := registry.ResolveVersion(ctx, ref)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolving reference:", err)
		os.Exit(1)
	}

	latest := "none"
	if len(orb.Versions) > 0 {
		latest = orb.Versions[0].Version
	}

	fmt.Printf("host:     %s\n", host)
	fmt.Printf("orb:      %s (%s)\n", orb.Name, orb.ID)
	fmt.Printf("resolved: %s -> %s (%s)\n", ref, resolved.Version, resolved.ID)
	fmt.Printf("versions: %d published, latest %s\n", len(orb.Versions), latest)
	fmt.Printf("siblings: %d alongside the resolved version\n", len(resolved.Versions))
	fmt.Printf("source:   %d bytes\n", len(resolved.Source))
}
