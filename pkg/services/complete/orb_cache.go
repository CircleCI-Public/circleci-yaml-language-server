package complete

import (
	"fmt"
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
)

const versionCount = 10000

type OrbCache struct {
	mutex        sync.Mutex
	registryOrbs map[string]*NamespaceOrbResponse
	orbData      map[string]*OrbGQLData
}

type OrbGQLData struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Versions []OrbVersion `json:"versions"`
}

type OrbVersion struct {
	Version string `json:"version"`
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

type RequestConfig struct {
	HostUrl  string
	Token    string
	UserId   string
	Query    string
	Params   map[string]interface{}
	Response interface{}
}

func (cache *OrbCache) request(config RequestConfig) (err error) {
	fmt.Printf("config = %+v\n", config)
	client := utils.NewClient(
		config.HostUrl,
		"graphql-unstable",
		config.Token,
		false,
	)
	request := utils.NewRequest(config.Query)
	request.SetToken(client.Token)
	request.SetUserId(config.UserId)
	for key, value := range config.Params {
		request.Var(key, value)
	}

	err = client.Run(request, config.Response)

	return
}

func (cache *OrbCache) GetOrbsOfRegistry(registry, hostUrl, token, userId string) (*NamespaceOrbResponse, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// If data is cached return it
	cached, cacheExists := cache.registryOrbs[registry]
	if cacheExists {
		return cached, nil
	}

	// Else request it
	query := `
	query OrbsByRegistry($name: String!, $versionCount: Int!) {
			registryNamespace(name: $name) {
				orbs(first: 1000){
					edges {
						cursor
						node {
							id
							name
							versions(count: $versionCount) {
								version
							}
						}
					}
				}
			}
		}
	`
	var response NamespaceOrbResponse
	err := cache.request(RequestConfig{
		HostUrl: hostUrl,
		Token:   token,
		UserId:  userId,
		Query:   query,
		Params: map[string]interface{}{
			"name":         registry,
			"versionCount": versionCount,
		},
		Response: &response,
	})
	if err != nil {
		return nil, err
	}

	// Then cache it and return it
	cache.registryOrbs[registry] = &response
	for i, orb := range response.RegistryNamespace.Orbs.Edges {
		// Here we point to response.RegistryNamespace.Orbs.Edges[i].Node and not to orb.Node.Name
		// because orb.Node.Name points to the loop-local variable orb and not to the real data of the
		// response struct, thus creating pointer errors
		cache.orbData[orb.Node.Name] = &response.RegistryNamespace.Orbs.Edges[i].Node
	}

	return &response, nil
}

func (cache *OrbCache) GetVersionsOfOrb(orbName, hostUrl, token, userId string) (*OrbGQLData, error) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	// If data is cached return it
	cached, cacheExists := cache.orbData[orbName]
	if cacheExists {
		return cached, nil
	}

	// Else request it
	query := `
	query OrbVersions($name: String!, $versionCount: Int!) {
			orb(name: $name) {
				name
				versions(count: $versionCount) {
					version
				}
			}
		}
	`
	response := map[string]OrbGQLData{}
	err := cache.request(RequestConfig{
		HostUrl: hostUrl,
		Token:   token,
		UserId:  userId,
		Query:   query,
		Params: map[string]interface{}{
			"name":         orbName,
			"versionCount": versionCount,
		},
		Response: &response,
	})
	if err != nil {
		return nil, err
	}

	orb, ok := response["orb"]
	if !ok {
		return nil, fmt.Errorf("No orb found")
	}

	// Then store it in the cache and return it
	cache.orbData[orbName] = &orb

	return &orb, nil
}
