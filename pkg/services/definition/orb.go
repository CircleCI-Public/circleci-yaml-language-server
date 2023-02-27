package definition

import (
	"fmt"
	"strings"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (def DefinitionStruct) getOrbDefinition() ([]protocol.Location, error) {
	var orb ast.Orb
	for _, currentOrb := range def.Doc.Orbs {
		if utils.PosInRange(currentOrb.NameRange, def.Params.Position) ||
			utils.PosInRange(currentOrb.Range, def.Params.Position) {
			orb = currentOrb
		}
	}

	orbInfo, err := def.GetOrbInfo(orb.Name)

	if orb.Url.IsLocal {
		return DefinitionStruct{
			Cache:  def.Cache,
			Params: def.Params,
			Doc:    def.Doc.FromOrbParsedAttributesToYamlDocument(orbInfo.OrbParsedAttributes),
		}.Definition()
	}

	if err != nil {
		return nil, err
	}

	if orbInfo == nil {
		return []protocol.Location{}, nil
	}

	return []protocol.Location{
		{
			URI:   uri.New(orbInfo.RemoteInfo.FilePath),
			Range: protocol.Range{},
		},
	}, nil
}

func (def DefinitionStruct) getOrbLocation(name string, redirectToOrbFile bool) ([]protocol.Location, error) {
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

			return []protocol.Location{
				{
					Range: orb.Range,
					URI:   def.Doc.URI,
				},
			}, nil
		}
	}

	return []protocol.Location{}, fmt.Errorf("orb not found")
}

func (def DefinitionStruct) getOrbCommandOrJobLocation(orbInfo *ast.OrbInfo, name string) ([]protocol.Location, error) {
	var fileUri protocol.DocumentURI

	if orbInfo.IsLocal {
		fileUri = def.Doc.URI
	} else {
		fileUri = uri.New(orbInfo.RemoteInfo.FilePath)
	}

	command, ok := orbInfo.Commands[name]
	if ok {
		return []protocol.Location{
			{
				URI:   fileUri,
				Range: command.Range,
			},
		}, nil
	}

	job, ok := orbInfo.Jobs[name]
	if ok {
		return []protocol.Location{
			{
				URI:   fileUri,
				Range: job.Range,
			},
		}, nil
	}

	return []protocol.Location{}, fmt.Errorf("orb command or job not found")
}

func (def DefinitionStruct) getOrbParamLocation(name string, paramName string) ([]protocol.Location, error) {
	splittedName := strings.Split(name, "/")
	if len(splittedName) < 2 {
		return []protocol.Location{}, fmt.Errorf("orb not found")
	}

	orbFile, err := def.GetOrbInfo(name)

	if err != nil {
		return []protocol.Location{}, err
	}

	if orbFile == nil {
		return []protocol.Location{}, fmt.Errorf("orb not found")
	}

	return def.getOrbCommandOrJobParamLocation(orbFile, splittedName[1], paramName)
}

func (def DefinitionStruct) getOrbCommandOrJobParamLocation(orbFile *ast.OrbInfo, name string, paramName string) ([]protocol.Location, error) {
	var fileUri protocol.DocumentURI

	if orbFile.IsLocal {
		fileUri = def.Doc.URI
	} else {
		fileUri = uri.New(orbFile.RemoteInfo.FilePath)
	}

	orbCommand, ok := orbFile.Commands[name]
	if ok {
		if param, ok := orbCommand.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					URI:   fileUri,
					Range: param.GetRange(),
				},
			}, nil
		}
	}

	orbJob, ok := orbFile.Jobs[name]
	if ok {
		if param, ok := orbJob.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					URI:   fileUri,
					Range: param.GetRange(),
				},
			}, nil
		}
	}

	return []protocol.Location{}, fmt.Errorf("orb command or job not found")
}
