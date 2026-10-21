package main

import (
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// config is a file to check, named by its slash-separated path under the
// directory it was found in, so that runs from different checkouts and
// platforms compare.
type config struct {
	dir  string
	name string
}

func (c config) path() string {
	return filepath.Join(c.dir, filepath.FromSlash(c.name))
}

// referenceConfigs are the build.yml files of the reference corpus.
func referenceConfigs(dir string) ([]config, error) {
	return walk(dir, func(name string) bool {
		return path.Base(name) == "build.yml"
	})
}

// repoConfigs are the YAML files under the repo's testdata directories, and
// those survey fetched into bin/survey.
func repoConfigs(dir string) ([]config, error) {
	return walk(dir, func(name string) bool {
		if !isYAML(name) {
			return false
		}
		return slices.Contains(strings.Split(name, "/"), "testdata") || strings.HasPrefix(name, "bin/survey/")
	})
}

func walk(dir string, keep func(name string) bool) ([]config, error) {
	var res []config
	err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		if keep(name) {
			res = append(res, config{dir, name})
		}
		return nil
	})
	return res, err
}

func isYAML(name string) bool {
	return strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")
}
