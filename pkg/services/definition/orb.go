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
	orbId := ""
	for _, orb := range def.Doc.Orbs {
		if utils.PosInRange(orb.NameRange, def.Params.Position) {
			orbId = orb.Url.GetOrbID()
		}
	}

	if orb := def.Cache.OrbCache.GetOrb(orbId); orb != nil {
		return []protocol.Location{
			{
				URI:   uri.New(orb.RemoteInfo.FilePath),
				Range: protocol.Range{},
			},
		}, nil
	}

	return []protocol.Location{}, nil
}

func (def DefinitionStruct) getOrbLocation(name string, redirectToOrbFile bool) ([]protocol.Location, error) {
	splittedName := strings.Split(name, "/")
	if len(splittedName) >= 2 {
		if orb, ok := def.Doc.Orbs[splittedName[0]]; ok {

			if redirectToOrbFile {
				orbFile := def.Cache.OrbCache.GetOrb(orb.Url.GetOrbID())
				return def.getOrbCommandOrJobLocation(*orbFile, splittedName[1])
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

func (def DefinitionStruct) getOrbCommandOrJobLocation(orbFile ast.CachedOrb, name string) ([]protocol.Location, error) {
	if orbCommand, ok := orbFile.Commands[name]; ok {
		return []protocol.Location{
			{
				URI:   uri.New(orbFile.RemoteInfo.FilePath),
				Range: orbCommand.Range,
			},
		}, nil
	}

	if orbJob, ok := orbFile.Jobs[name]; ok {
		return []protocol.Location{
			{
				URI:   uri.New(orbFile.RemoteInfo.FilePath),
				Range: orbJob.Range,
			},
		}, nil
	}

	return []protocol.Location{}, fmt.Errorf("orb command or job not found")
}

func (def DefinitionStruct) getOrbParamLocation(name string, paramName string) ([]protocol.Location, error) {
	splittedName := strings.Split(name, "/")
	if len(splittedName) >= 2 {
		if orb, ok := def.Doc.Orbs[splittedName[0]]; ok {

			orbFile := def.Cache.OrbCache.GetOrb(orb.Url.GetOrbID())
			return def.getOrbCommandOrJobParamLocation(*orbFile, splittedName[1], paramName)
		}
	}

	return []protocol.Location{}, fmt.Errorf("orb not found")
}

func (def DefinitionStruct) getOrbCommandOrJobParamLocation(orbFile ast.CachedOrb, name string, paramName string) ([]protocol.Location, error) {
	if orbCommand, ok := orbFile.Commands[name]; ok {
		if param, ok := orbCommand.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					URI:   uri.New(orbFile.RemoteInfo.FilePath),
					Range: param.GetRange(),
				},
			}, nil
		}
	}

	if orbJob, ok := orbFile.Jobs[name]; ok {
		if param, ok := orbJob.Parameters[paramName]; ok {
			return []protocol.Location{
				{
					URI:   uri.New(orbFile.RemoteInfo.FilePath),
					Range: param.GetRange(),
				},
			}, nil
		}
	}

	return []protocol.Location{}, fmt.Errorf("orb command or job not found")
}
