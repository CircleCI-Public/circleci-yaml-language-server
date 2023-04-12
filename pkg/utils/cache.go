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
	FileCache    FileCache
	OrbCache     OrbCache
	DockerCache  DockerCache
	ContextCache ContextCache
}

type DockerCache struct {
	cacheMutex  *sync.Mutex
	dockerCache map[string]*CachedDockerImage
}

type CachedDockerImage struct {
	Checked bool
	Exists  bool
}

type CachedFile struct {
	TextDocument protocol.TextDocumentItem
	Project      Project
	EnvVariables []string
}

type FileCache struct {
	cacheMutex *sync.Mutex
	fileCache  map[protocol.URI]*CachedFile
}

type OrbCache struct {
	cacheMutex *sync.Mutex
	orbsCache  map[string]*ast.OrbInfo
}

type ContextCache struct {
	cacheMutex   *sync.Mutex
	contextCache map[string]map[string]*Context
}

func (c *Cache) init() {
	c.FileCache.fileCache = make(map[protocol.URI]*CachedFile)
	c.FileCache.cacheMutex = &sync.Mutex{}

	c.OrbCache.orbsCache = make(map[string]*ast.OrbInfo)
	c.OrbCache.cacheMutex = &sync.Mutex{}

	c.DockerCache.cacheMutex = &sync.Mutex{}
	c.DockerCache.dockerCache = make(map[string]*CachedDockerImage)

	c.ContextCache.cacheMutex = &sync.Mutex{}
	c.ContextCache.contextCache = make(map[string]map[string]*Context)
}

// FILE

func (c *FileCache) SetFile(cachedFile CachedFile) CachedFile {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.fileCache[cachedFile.TextDocument.URI] = &cachedFile
	return cachedFile
}

func (c *FileCache) GetFile(uri protocol.URI) *CachedFile {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache[uri]
}

func (c *FileCache) GetFiles() map[protocol.URI]*CachedFile {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache
}

func (c *FileCache) RemoveFile(uri protocol.URI) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.fileCache, uri)
}

func (c *FileCache) AddEnvVariableToProjectLinkedToFile(uri protocol.URI, envVariable string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	project := c.fileCache[uri]

	if FindInArray(project.EnvVariables, envVariable) < 0 {
		project.EnvVariables = append(project.EnvVariables, envVariable)
	}
	c.fileCache[uri] = project
}

func (c *FileCache) AddProjectSlugToFile(uri protocol.URI, project Project) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file := c.fileCache[uri]

	file.Project = project

	c.fileCache[uri] = file
}

func (c *FileCache) UpdateTextDocument(uri protocol.URI, textDocument protocol.TextDocumentItem) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file := c.fileCache[uri]
	file.TextDocument = textDocument

	c.fileCache[uri] = file
}

// ORBS

func (c *OrbCache) HasOrb(orbID string) bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	_, ok := c.orbsCache[orbID]

	return ok
}

func (c *OrbCache) SetOrb(orb *ast.OrbInfo, orbID string) ast.OrbInfo {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.orbsCache[orbID] = orb
	return *orb
}

func (c *OrbCache) UpdateOrbParsedAttributes(orbID string, parsedOrbAttributes ast.OrbParsedAttributes) ast.OrbParsedAttributes {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.orbsCache[orbID].OrbParsedAttributes = parsedOrbAttributes
	return parsedOrbAttributes
}

func (c *OrbCache) GetOrb(orbID string) *ast.OrbInfo {
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
		if _, err := os.Stat(orb.RemoteInfo.FilePath); err == nil {
			os.Remove(orb.RemoteInfo.FilePath)
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

func (cache *Cache) ClearHostData() {
	cache.RemoveOrbFiles()
	cache.OrbCache.RemoveOrbs()
}

// Context cache

func (c *ContextCache) SetOrganizationContext(organizationId string, ctx *Context) *Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if c.contextCache[organizationId] == nil {
		c.contextCache[organizationId] = make(map[string]*Context)
	}
	c.contextCache[organizationId][ctx.Name] = ctx
	return ctx
}

func (c *ContextCache) GetOrganizationContext(organizationId string, name string) *Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.contextCache[organizationId][name]
}

func (c *ContextCache) RemoveOrganizationContext(organizationId string, name string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	org := c.contextCache[organizationId]
	delete(org, name)
}

func (c *ContextCache) AddEnvVariableToOrganizationContext(organizationId string, name string, envVariable string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	ctx := c.contextCache[organizationId][name]

	if FindInArray(ctx.envVariables, envVariable) < 0 {
		ctx.envVariables = append(ctx.envVariables, envVariable)
	}
	c.contextCache[organizationId][name] = ctx
}

func (c *ContextCache) GetAllContextOfOrganization(organizationId string) map[string]*Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.contextCache[organizationId]
}
