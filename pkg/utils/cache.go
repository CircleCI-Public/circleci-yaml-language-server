package utils

import (
	"os"
	"path"
	"sync"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/adrg/xdg"
	"go.lsp.dev/protocol"
)

type Cache struct {
	FileCache   FileCache
	OrbCache    OrbCache
	DockerCache DockerCache
	TokenCache  TokenCache
}

type TokenCache struct {
	cacheMutex *sync.Mutex
	token      string
}

type DockerCache struct {
	cacheMutex  *sync.Mutex
	dockerCache map[string]*CachedDockerImage
}

type CachedDockerImage struct {
	Checked bool
	Exists  bool
}

type FileCache struct {
	cacheMutex *sync.Mutex
	fileCache  map[protocol.URI]*protocol.TextDocumentItem
}

type OrbCache struct {
	cacheMutex *sync.Mutex
	orbsCache  map[string]*ast.CachedOrb
}

func (c *Cache) init() {
	c.FileCache.fileCache = make(map[protocol.URI]*protocol.TextDocumentItem)
	c.FileCache.cacheMutex = &sync.Mutex{}

	c.OrbCache.orbsCache = make(map[string]*ast.CachedOrb)
	c.OrbCache.cacheMutex = &sync.Mutex{}

	c.DockerCache.cacheMutex = &sync.Mutex{}
	c.DockerCache.dockerCache = make(map[string]*CachedDockerImage)
	c.TokenCache.cacheMutex = &sync.Mutex{}
	c.TokenCache.token = ""
}

// FILE

func (c *FileCache) SetFile(file *protocol.TextDocumentItem) protocol.TextDocumentItem {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.fileCache[file.URI] = file
	return *file
}

func (c *FileCache) GetFile(uri protocol.URI) *protocol.TextDocumentItem {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache[uri]
}

func (c *FileCache) GetFiles() map[protocol.URI]*protocol.TextDocumentItem {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache
}

func (c *FileCache) RemoveFile(uri protocol.URI) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.fileCache, uri)
}

// ORBS

func (c *OrbCache) HasOrb(orbID string) bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	_, ok := c.orbsCache[orbID]

	return ok
}

func (c *OrbCache) SetOrb(orb *ast.CachedOrb, orbID string) ast.CachedOrb {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.orbsCache[orbID] = orb
	return *orb
}

func (c *TokenCache) SetToken(token string) string {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.token = token
	return token
}

func (c *TokenCache) GetToken() string {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.token
}

func (c *OrbCache) GetOrb(orbID string) *ast.CachedOrb {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.orbsCache[orbID]
}

func (c *OrbCache) RemoveOrb(orbID string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.orbsCache, orbID)
}

func (c *OrbCache) RemoveOrbs() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	for k := range c.orbsCache {
		delete(c.orbsCache, k)
	}
}

func (c *Cache) RemoveOrbFiles() {
	c.OrbCache.cacheMutex.Lock()
	defer c.OrbCache.cacheMutex.Unlock()
	c.FileCache.cacheMutex.Lock()
	defer c.FileCache.cacheMutex.Unlock()

	for _, orb := range c.OrbCache.orbsCache {
		if _, err := os.Stat(orb.FilePath); err == nil {
			os.Remove(orb.FilePath)
		}
	}
}

// Docker images cache

func (c *DockerCache) Add(name string, exists bool) *CachedDockerImage {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.dockerCache[name] = &CachedDockerImage{
		Checked: true,
		Exists:  exists,
	}

	return c.dockerCache[name]
}

func (c *DockerCache) Get(name string) *CachedDockerImage {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	return c.dockerCache[name]
}

func (c *DockerCache) Remove(name string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	delete(c.dockerCache, name)
}

func CreateCache() *Cache {
	cache := Cache{}
	cache.init()
	return &cache
}

func GetOrbCacheFSPath(orbYaml string) string {
	file := path.Join("cci", "orbs", ".circleci", orbYaml+".yml")
	filePath, err := xdg.CacheFile(file)

	if err != nil {
		filePath = path.Join(xdg.Home, ".cache", file)
	}

	return filePath
}
