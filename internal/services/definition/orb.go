package definition

import (
	"fmt"
	"strings"

	"go.lsp.dev/uri"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) getOrbDefinition() ([]Link, error) {
	var orb ast.Orb
	for _, currentOrb := range def.Doc.Orbs {
		if position.InRange(currentOrb.NameRange, def.Params.Position) ||
			position.InRange(currentOrb.Range, def.Params.Position) {
			orb = currentOrb
		}
	}

	orbInfo, err := def.GetOrbInfo(orb.Name)

	if orb.Url.IsLocal {
		return DefinitionStruct{
			Cache:  def.Cache,
			Params: def.Params,
			Doc:    def.Doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes),
		}.search()
	}

	if err != nil {
		return nil, err
	}

	if orbInfo == nil {
		return []Link{}, nil
	}

	return []Link{{URI: uri.File(orbInfo.RemoteInfo.FilePath)}}, nil
}

func (def DefinitionStruct) getOrbLocation(name string, redirectToOrbFile bool) ([]Link, error) {
	splittedName := strings.Split(name, "/")
	if len(splittedName) >= 2 {
		if orb, ok := def.Doc.Orbs[splittedName[0]]; ok {

			if redirectToOrbFile {
				orbFile, err := def.GetOrbInfo(orb.Name)

				if err != nil {
					return nil, err
				}

				return def.getOrbCommandOrJobLocation(orbFile, splittedName[1])
			}

			return []Link{{URI: def.Doc.URI, Range: orb.Range, NameRange: orb.NameRange}}, nil
		}
	}

	return []Link{}, fmt.Errorf("orb not found")
}

func (def DefinitionStruct) getOrbCommandOrJobLocation(orbInfo *ast.OrbInfo, name string) ([]Link, error) {
	var fileUri uri.URI

	if orbInfo.IsLocal {
		fileUri = def.Doc.URI
	} else {
		fileUri = uri.File(orbInfo.RemoteInfo.FilePath)
	}

	command, ok := orbInfo.Commands[name]
	if ok {
		return []Link{{URI: fileUri, Range: command.Range, NameRange: command.NameRange}}, nil
	}

	job, ok := orbInfo.Jobs[name]
	if ok {
		return []Link{{URI: fileUri, Range: job.Range, NameRange: job.NameRange}}, nil
	}

	return []Link{}, fmt.Errorf("orb command or job not found")
}

func (def DefinitionStruct) getOrbParamLocation(name string, paramName string) ([]Link, error) {
	splittedName := strings.Split(name, "/")
	if len(splittedName) < 2 {
		return []Link{}, fmt.Errorf("orb not found")
	}

	orbName := splittedName[0]
	orbFile, err := def.GetOrbInfo(orbName)

	if err != nil {
		return []Link{}, err
	}

	if orbFile == nil {
		return []Link{}, fmt.Errorf("orb not found")
	}

	return def.getOrbCommandOrJobParamLocation(orbFile, splittedName[1], paramName)
}

func (def DefinitionStruct) getOrbCommandOrJobParamLocation(orbFile *ast.OrbInfo, name string, paramName string) ([]Link, error) {
	var fileUri uri.URI

	if orbFile.IsLocal {
		fileUri = def.Doc.URI
	} else {
		fileUri = uri.File(orbFile.RemoteInfo.FilePath)
	}

	orbCommand, ok := orbFile.Commands[name]
	if ok {
		if param, ok := orbCommand.Parameters[paramName]; ok {
			return []Link{{URI: fileUri, Range: param.GetRange(), NameRange: param.GetNameRange()}}, nil
		}
	}

	orbJob, ok := orbFile.Jobs[name]
	if ok {
		if param, ok := orbJob.Parameters[paramName]; ok {
			return []Link{{URI: fileUri, Range: param.GetRange(), NameRange: param.GetNameRange()}}, nil
		}
	}

	return []Link{}, fmt.Errorf("orb command or job not found")
}
