package complete

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	sitter "github.com/smacker/go-tree-sitter"
)

var simpleRegistryOrbsCache = make(map[string]*NamespaceOrbResponse)
var fullOrbNameRegex = regexp.MustCompile(`^([a-z0-9\-_]+)\/([a-z0-9\-_]+)@(.*)$`)
var registryAndNameRegex = regexp.MustCompile(`^([a-z0-9\-_]+)\/([a-z0-9\-_]+)$`)

func (ch *CompletionHandler) completeOrbs() {
	if ch.DocTag != "original" {
		return
	}

	for _, orb := range ch.Doc.Orbs {
		if orb.ValueNode == nil || !utils.PosInRange(orb.ValueRange, ch.Params.Position) || orb.ValueNode.Type() != "flow_node" {
			continue
		}

		child := parser.GetFirstChild(orb.ValueNode)

		if child == nil {
			continue
		}

		ch.completeOrbName(child)

		return
	}
}

func (ch *CompletionHandler) completeOrbName(node *sitter.Node) {
	completions, err := getOrbNameCompletions(
		ch.Doc.GetNodeText(node),
		ch.Doc.Context.Api.HostUrl,
		ch.Doc.Context.Api.Token,
		ch.Doc.Context.UserId,
	)

	if err != nil {
		return
	}

	for _, completion := range completions {
		ch.addReplaceTextCompletionItem(node, completion)
	}
}

func getOrbNameCompletions(name, hostUrl, token, userId string) ([]string, error) {
	parts := strings.Split(name, "/")
	registry := parts[0]

	response, err := fetchOrbsByRegistry(registry, hostUrl, token, userId)

	if err != nil {
		return nil, err
	}

	completions := make([]string, len(response.RegistryNamespace.Orbs.Edges))

	for i, v := range response.RegistryNamespace.Orbs.Edges {
		if len(v.Node.Versions) > 0 {
			completions[i] = fmt.Sprintf("%s@%s", v.Node.Name, v.Node.Versions[0].Version)
		}
	}

	return completions, nil
}

func fetchOrbsByRegistry(registry, hostUrl, token, userId string) (*NamespaceOrbResponse, error) {
	cached, cacheExists := simpleRegistryOrbsCache[registry]

	if cacheExists {
		return cached, nil
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}

	client := utils.NewClient(
		httpClient,
		hostUrl,
		"graphql-unstable",
		token,
		false,
	)

	query := `
		query OrbsByRegistry($name: String!) {
			registryNamespace(name: $name) {
				orbs(first: 1000){
					edges {
						cursor
						node {
							id
							name
							versions(count: 10) {version}
						}
					}
				}
			}
		}
	`

	request := utils.NewRequest(query)
	request.SetToken(client.Token)
	request.SetUserId(userId)
	request.Var("name", registry)

	var response NamespaceOrbResponse
	err := client.Run(request, &response)

	if err != nil {
		return nil, err
	}

	simpleRegistryOrbsCache[registry] = &response

	return &response, nil
}

type OrbGQLData struct {
	ID       string
	Name     string
	Versions []struct {
		Version string `json:"version"`
		Source  string `json:"source"`
	} `json:"versions"`
}

type NamespaceOrbResponse struct {
	RegistryNamespace struct {
		ID   string
		Name string
		Orbs struct {
			Edges []struct {
				Cursor string
				Node   OrbGQLData
			}
			TotalCount int
			PageInfo   struct {
				HasNextPage bool
			}
		}
	}
}
