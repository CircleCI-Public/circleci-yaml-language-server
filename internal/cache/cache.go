package cache

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"go.lsp.dev/protocol"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
)

type Cache struct {
	FileCache             Files
	OrbCache              Orbs
	DockerCache           DockerImages
	DockerTagsCache       DockerTags
	ResourceClassCache    ResourceClasses
	ContextCache          Contexts
	MachineOfferingsCache MachineOfferings
}

type DockerImages struct {
	cacheMutex  *sync.Mutex
	dockerCache map[string]*DockerImage
}

type DockerImage struct {
	Checked bool
	Exists  bool
}

type ImageTags struct {
	// A tag to recommended for untagged images
	Recommended string

	// The key of the map are the tag checked and the value is whether this tag exists or not
	CheckedTags map[string]bool
}

type DockerTags struct {
	cacheMutex *sync.Mutex
	tagsCache  map[string]ImageTags
}

type File struct {
	TextDocument protocol.TextDocumentItem
	Project      circleci.Project
	EnvVariables []string
}

type Files struct {
	cacheMutex *sync.Mutex
	fileCache  map[protocol.URI]*File
}

type Orbs struct {
	cacheMutex *sync.Mutex
	orbsCache  map[string]*ast.OrbInfo
}

type Contexts struct {
	cacheMutex          *sync.Mutex
	contextCache        map[string]map[string]*Context
	ambiguousShortNames map[string]map[string]struct{}
	listLoadedOrgs      map[string]bool
}

type ResourceClasses struct {
	cacheMutex         *sync.Mutex
	resourceClassCache map[protocol.URI]*[]string
}

func (c *Cache) init() {
	c.FileCache.fileCache = make(map[protocol.URI]*File)
	c.FileCache.cacheMutex = &sync.Mutex{}

	c.OrbCache.orbsCache = make(map[string]*ast.OrbInfo)
	c.OrbCache.cacheMutex = &sync.Mutex{}

	c.DockerCache.cacheMutex = &sync.Mutex{}
	c.DockerCache.dockerCache = make(map[string]*DockerImage)

	c.DockerTagsCache.cacheMutex = &sync.Mutex{}
	c.DockerTagsCache.tagsCache = make(map[string]ImageTags)

	c.ContextCache.cacheMutex = &sync.Mutex{}
	c.ContextCache.contextCache = make(map[string]map[string]*Context)
	c.ContextCache.ambiguousShortNames = make(map[string]map[string]struct{})
	c.ContextCache.listLoadedOrgs = make(map[string]bool)

	c.ResourceClassCache.cacheMutex = &sync.Mutex{}
	c.ResourceClassCache.resourceClassCache = make(map[protocol.URI]*[]string)
}

// FILE

func (c *Files) SetFile(cachedFile File) File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.fileCache[cachedFile.TextDocument.URI] = &cachedFile
	return cachedFile
}

func (c *Files) GetFile(uri protocol.URI) *File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache[uri]
}

func (c *Files) GetFiles() map[protocol.URI]*File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.fileCache
}

func (c *Files) RemoveFile(uri protocol.URI) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.fileCache, uri)
}

func (c *Files) AddEnvVariableToProjectLinkedToFile(uri protocol.URI, envVariable string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	project := c.fileCache[uri]

	if !slices.Contains(project.EnvVariables, envVariable) {
		project.EnvVariables = append(project.EnvVariables, envVariable)
	}
	c.fileCache[uri] = project
}

func (c *Files) AddProjectSlugToFile(uri protocol.URI, project circleci.Project) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file := c.fileCache[uri]

	file.Project = project

	c.fileCache[uri] = file
}

// forgetProjects drops the project resolved for every file, and the variables
// read for it, so that each is resolved again on next use.
func (c *Files) forgetProjects() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	for _, file := range c.fileCache {
		file.Project = circleci.Project{}
		file.EnvVariables = nil
	}
}

func (c *Files) UpdateTextDocument(uri protocol.URI, textDocument protocol.TextDocumentItem) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file := c.fileCache[uri]
	file.TextDocument = textDocument

	c.fileCache[uri] = file
}

// ORBS

func (c *Orbs) HasOrb(orbID string) bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	_, ok := c.orbsCache[orbID]

	return ok
}

func (c *Orbs) SetOrb(orb *ast.OrbInfo, orbID string) ast.OrbInfo {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.orbsCache[orbID] = orb
	return *orb
}

func (c *Orbs) UpdateOrbParsedAttributes(orbID string, parsedOrbAttributes ast.OrbParsedAttributes) ast.OrbParsedAttributes {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.orbsCache[orbID].OrbParsedAttributes = parsedOrbAttributes
	return parsedOrbAttributes
}

func (c *Orbs) GetOrb(orbID string) *ast.OrbInfo {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.orbsCache[orbID]
}

func (c *Orbs) RemoveOrb(orbID string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.orbsCache, orbID)
}

func (c *Orbs) RemoveOrbs() {
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
			_ = os.Remove(orb.RemoteInfo.FilePath)
		}
	}
}

// Docker images cache

func (c *DockerImages) Add(name string, exists bool) *DockerImage {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.dockerCache[name] = &DockerImage{
		Checked: true,
		Exists:  exists,
	}

	return c.dockerCache[name]
}

func (c *DockerImages) Get(name string) *DockerImage {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	return c.dockerCache[name]
}

func (c *DockerImages) Remove(name string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	delete(c.dockerCache, name)
}

// Docker tags cache

func (c *DockerTags) Add(namespace, image string, value ImageTags) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	c.tagsCache[fmt.Sprintf("%s/%s", namespace, image)] = value
}

func (c *DockerTags) Get(namespace, image string) *ImageTags {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	tags, ok := c.tagsCache[fmt.Sprintf("%s/%s", namespace, image)]
	if !ok {
		return nil
	}
	return &tags
}

// Cache

func New() *Cache {
	cache := Cache{}
	cache.init()
	return &cache
}

func OrbSourcePath(orbYaml string) string {
	file := path.Join("cci", "orbs", ".circleci", orbYaml+".yml")
	filePath, err := xdg.CacheFile(file)
	if err != nil {
		filePath = path.Join(xdg.Home, ".cache", file)
	}

	return filePath
}

// ClearHostData forgets everything read from the API host, for when the host
// or the token changes. A file's project is among it: it was resolved against
// the old host, and its organization id is what contexts are looked up by.
func (cache *Cache) ClearHostData() {
	cache.RemoveOrbFiles()
	cache.OrbCache.RemoveOrbs()
	cache.clearContextCache()
	cache.FileCache.forgetProjects()
}

func (cache *Cache) clearContextCache() {
	cache.ContextCache.cacheMutex.Lock()
	defer cache.ContextCache.cacheMutex.Unlock()
	cache.ContextCache.contextCache = make(map[string]map[string]*Context)
	cache.ContextCache.ambiguousShortNames = make(map[string]map[string]struct{})
	cache.ContextCache.listLoadedOrgs = make(map[string]bool)
}

func (c *Contexts) MarkOrganizationContextListLoaded(organizationId string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if c.listLoadedOrgs == nil {
		c.listLoadedOrgs = make(map[string]bool)
	}
	c.listLoadedOrgs[organizationId] = true
}

func (c *Contexts) IsOrganizationContextListLoaded(organizationId string) bool {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.listLoadedOrgs[organizationId]
}

func (c *Contexts) ClearOrganizationContextList(organizationId string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.contextCache, organizationId)
	delete(c.ambiguousShortNames, organizationId)
	delete(c.listLoadedOrgs, organizationId)
}

func (c *Contexts) SetOrganizationContext(organizationId string, ctx *Context) *Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	if c.contextCache[organizationId] == nil {
		c.contextCache[organizationId] = make(map[string]*Context)
	}
	orgMap := c.contextCache[organizationId]
	orgMap[ctx.Name] = ctx
	// CircleCI configs often use the short context name for the project's org; the API may
	// return a qualified name (org/context). Also index by the suffix when unambiguous.
	if i := strings.LastIndex(ctx.Name, "/"); i >= 0 {
		short := ctx.Name[i+1:]
		if short == "" {
			return ctx
		}
		if c.isAmbiguousShortName(organizationId, short) {
			return ctx
		}
		if existing, exists := orgMap[short]; exists {
			if existing != ctx {
				delete(orgMap, short)
				c.markAmbiguousShortName(organizationId, short)
			}
			return ctx
		}
		orgMap[short] = ctx
	}
	return ctx
}

func (c *Contexts) isAmbiguousShortName(organizationId, short string) bool {
	ambiguous, ok := c.ambiguousShortNames[organizationId]
	if !ok {
		return false
	}
	_, ok = ambiguous[short]
	return ok
}

func (c *Contexts) markAmbiguousShortName(organizationId, short string) {
	if c.ambiguousShortNames[organizationId] == nil {
		c.ambiguousShortNames[organizationId] = make(map[string]struct{})
	}
	c.ambiguousShortNames[organizationId][short] = struct{}{}
}

func (c *Contexts) GetOrganizationContext(organizationId string, name string) *Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.contextCache[organizationId][name]
}

// ResolveWorkflowContext looks up a context as referenced in config: exact key, then common
// alternates between short vs org-qualified names (API and config do not always agree).
func (c *Contexts) ResolveWorkflowContext(organizationId, organizationSlug, workflowContextName string) *Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	org := c.contextCache[organizationId]
	if org == nil {
		return nil
	}
	if ctx := org[workflowContextName]; ctx != nil {
		return ctx
	}
	if i := strings.LastIndex(workflowContextName, "/"); i >= 0 {
		short := workflowContextName[i+1:]
		if short != "" {
			if ctx := org[short]; ctx != nil {
				return ctx
			}
		}
	}
	if organizationSlug != "" && !strings.Contains(workflowContextName, "/") {
		if ctx := org[organizationSlug+"/"+workflowContextName]; ctx != nil {
			return ctx
		}
	}
	return nil
}

func (c *Contexts) RemoveOrganizationContext(organizationId string, name string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	org := c.contextCache[organizationId]
	delete(org, name)
}

func (c *Contexts) AddEnvVariableToOrganizationContext(organizationId string, name string, envVariable string) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	ctx := c.contextCache[organizationId][name]

	if !slices.Contains(ctx.envVariables, envVariable) {
		ctx.envVariables = append(ctx.envVariables, envVariable)
	}
	c.contextCache[organizationId][name] = ctx
}

func (c *Contexts) GetAllContextOfOrganization(organizationId string) map[string]*Context {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	return c.contextCache[organizationId]
}

// Resource class

func (c *ResourceClasses) SetResourceClassForFile(uri protocol.URI, resourceClass *[]string) *[]string {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.resourceClassCache[uri] = resourceClass
	return resourceClass
}

func (c *ResourceClasses) GetResourceClassOfFile(uri protocol.URI) []string {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	resourceClasses, ok := c.resourceClassCache[uri]
	if !ok {
		return []string{}
	}
	return *resourceClasses
}
