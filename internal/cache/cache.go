// Package cache holds what the server knows beyond the documents it is given:
// what it has read from the API host and from Docker Hub.
//
// Every lookup goes through a memo.Memo, so that concurrent callers share one
// fetch and a failure is not remembered. What was read from the API host is
// forgotten by ClearHostData when the host or the token changes.
package cache

import (
	"slices"
	"sync"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/client/circleci"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/memo"
)

type Cache struct {
	FileCache             Files
	OrbCache              Orbs
	OrbPackages           OrbPackages
	Functions             Functions
	DockerCache           DockerImages
	DockerTagsCache       DockerTags
	ResourceClassCache    ResourceClasses
	ContextCache          Contexts
	MachineOfferingsCache MachineOfferings
	NamespaceCache        Namespaces
	ProjectCache          Projects

	orbSources orbSources
}

// DockerImages remembers whether each Docker Hub repository exists, for
// memo.FoundLifetime or memo.NotFoundLifetime. The answer is kept per
// repository rather than per image reference: every tag of an image shares
// it.
type DockerImages struct {
	images *memo.Memo[bool]
}

type ImageTags struct {
	// A tag to recommended for untagged images
	Recommended string

	// The tags the listing found, each mapped to true. A tag missing here may
	// still exist beyond the page that was read; DockerTags.HasTag asks.
	// Shared between callers, so never written once cached.
	CheckedTags map[string]bool
}

// DockerTags remembers the tags listed for each Docker Hub repository, for
// memo.FoundLifetime, and the answer for each tag asked about on its own, for
// memo.FoundLifetime or memo.NotFoundLifetime.
type DockerTags struct {
	lists  *memo.Memo[ImageTags]
	checks *memo.Memo[bool]
}

type File struct {
	TextDocument protocol.TextDocumentItem
	Project      circleci.Project
	EnvVariables []string
}

type Files struct {
	cacheMutex *sync.Mutex
	fileCache  map[uri.URI]File
}

func (c *Cache) init() {
	c.FileCache.fileCache = make(map[uri.URI]File)
	c.FileCache.cacheMutex = &sync.Mutex{}

	c.OrbCache.orbs = memo.New(memo.Fixed[*ast.OrbInfo](memo.FoundLifetime), nil)
	c.OrbPackages.packages = memo.New(orbPackageLifetime, nil)
	c.OrbPackages.namespaces = memo.New(namespaceOrbsLifetime, nil)
	c.Functions.packages = memo.New(functionPackageLifetime, nil)
	c.Functions.descriptors = memo.New(memo.Fixed[*circleci.FunctionDescriptor](memo.FoundLifetime), nil)

	c.DockerCache.images = memo.New(memo.Existence, nil)
	c.DockerTagsCache.lists = memo.New(memo.Fixed[ImageTags](memo.FoundLifetime), nil)
	c.DockerTagsCache.checks = memo.New(memo.Existence, nil)
	c.NamespaceCache.namespaces = memo.New(memo.Existence, nil)

	c.ContextCache.orgs = memo.New(memo.Fixed[*orgContexts](listLifetime), nil)
	c.MachineOfferingsCache.catalog = memo.New(offeringsLifetime, nil)
	c.ProjectCache.projects = memo.New(projectLifetime, nil)

	c.ResourceClassCache.namespaceOfFile = make(map[uri.URI]string)
	c.ResourceClassCache.classes = memo.New(memo.Fixed[[]string](listLifetime), nil)
}

// listLifetime is how long a listing is kept: the contexts of an
// organization, or its runner resource classes. A listing answers whether
// each thing in it exists, and the usual fix for one that is missing is to
// create it, so it is kept only as long as an answer that something does not
// exist.
const listLifetime = memo.NotFoundLifetime

// FILE

// A file is handed out as a copy, and changed only by the methods here, under
// the lock: a validation reading a file in the background must not see it
// change half way through because the document was edited.

// copy returns a copy of a file that shares nothing with it.
func (file File) copy() *File {
	file.EnvVariables = slices.Clone(file.EnvVariables)
	return &file
}

func (c *Files) SetFile(cachedFile File) File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	c.fileCache[cachedFile.TextDocument.URI] = *cachedFile.copy()
	return cachedFile
}

func (c *Files) GetFile(uri uri.URI) *File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file, ok := c.fileCache[uri]
	if !ok {
		return nil
	}
	return file.copy()
}

func (c *Files) GetFiles() map[uri.URI]*File {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	files := make(map[uri.URI]*File, len(c.fileCache))
	for uri, file := range c.fileCache {
		files[uri] = file.copy()
	}
	return files
}

func (c *Files) RemoveFile(uri uri.URI) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	delete(c.fileCache, uri)
}

// update changes the file cached for uri, if there is one: a file closed while
// something was being fetched for it stays closed.
func (c *Files) update(uri uri.URI, change func(*File)) {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()
	file, ok := c.fileCache[uri]
	if !ok {
		return
	}
	change(&file)
	c.fileCache[uri] = file
}

func (c *Files) AddEnvVariableToProjectLinkedToFile(uri uri.URI, envVariable string) {
	c.update(uri, func(file *File) {
		if !slices.Contains(file.EnvVariables, envVariable) {
			file.EnvVariables = append(file.EnvVariables, envVariable)
		}
	})
}

// ClearEnvVariables forgets the variables read for a file's project.
func (c *Files) ClearEnvVariables(uri uri.URI) {
	c.update(uri, func(file *File) {
		file.EnvVariables = []string{}
	})
}

func (c *Files) AddProjectSlugToFile(uri uri.URI, project circleci.Project) {
	c.update(uri, func(file *File) {
		file.Project = project
	})
}

// forgetProjects drops the project resolved for every file, and the variables
// read for it, so that each is resolved again on next use.
func (c *Files) forgetProjects() {
	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	for uri, file := range c.fileCache {
		file.Project = circleci.Project{}
		file.EnvVariables = nil
		c.fileCache[uri] = file
	}
}

func (c *Files) UpdateTextDocument(uri uri.URI, textDocument protocol.TextDocumentItem) {
	c.update(uri, func(file *File) {
		file.TextDocument = textDocument
	})
}

// Docker images cache

func imageKey(namespace, image string) string {
	return namespace + "/" + image
}

// Exists reports whether an image exists, calling check only when no answer
// is remembered. An error from check is returned but not remembered.
func (c *DockerImages) Exists(namespace, image string, check func() (bool, error)) (bool, error) {
	return c.images.Get(imageKey(namespace, image), check)
}

func (c *DockerImages) Add(namespace, image string, exists bool) {
	c.images.Put(imageKey(namespace, image), exists)
}

// Get returns whether an image exists, and whether that is known at all.
func (c *DockerImages) Get(namespace, image string) (exists, known bool) {
	return c.images.Peek(imageKey(namespace, image))
}

// Docker tags cache

// Load returns the tags of an image, calling list only when none are
// remembered. An error from list is returned but not remembered.
func (c *DockerTags) Load(namespace, image string, list func() (ImageTags, error)) (ImageTags, error) {
	return c.lists.Get(imageKey(namespace, image), list)
}

func (c *DockerTags) Add(namespace, image string, value ImageTags) {
	c.lists.Put(imageKey(namespace, image), value)
}

func (c *DockerTags) Get(namespace, image string) *ImageTags {
	tags, ok := c.lists.Peek(imageKey(namespace, image))
	if !ok {
		return nil
	}
	return &tags
}

// HasTag reports whether an image has a tag, calling check only when no
// answer is remembered. An error from check is returned but not remembered.
func (c *DockerTags) HasTag(namespace, image, tag string, check func() (bool, error)) (bool, error) {
	return c.checks.Get(imageKey(namespace, image)+":"+tag, check)
}

// Checked returns whether an image has a tag, and whether that was asked
// about and answered on its own.
func (c *DockerTags) Checked(namespace, image, tag string) (exists, known bool) {
	return c.checks.Peek(imageKey(namespace, image) + ":" + tag)
}

// Cache

func New() *Cache {
	cache := Cache{}
	cache.init()
	return &cache
}

// ClearHostData forgets everything read from the API host, for when the host
// or the token changes. A file's project is among it: it was resolved against
// the old host, and its organization id is what contexts are looked up by.
//
// Docker Hub is not the API host, so what was read from it is kept.
func (cache *Cache) ClearHostData() {
	cache.OrbCache.orbs.Clear()
	cache.orbSources.remove()
	cache.OrbPackages.packages.Clear()
	cache.OrbPackages.namespaces.Clear()
	cache.Functions.packages.Clear()
	cache.Functions.descriptors.Clear()
	cache.ContextCache.orgs.Clear()
	cache.NamespaceCache.namespaces.Clear()
	cache.MachineOfferingsCache.catalog.Clear()
	cache.ResourceClassCache.classes.Clear()
	cache.ProjectCache.projects.Clear()
	cache.FileCache.forgetProjects()
}

// Close removes what the cache wrote to disk. The cache is still usable
// afterwards, and writes again as it needs to.
func (cache *Cache) Close() {
	cache.orbSources.remove()
}
