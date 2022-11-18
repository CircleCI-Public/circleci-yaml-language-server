package definition

import (
	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"github.com/circleci/circleci-yaml-language-server/pkg/utils"
	"go.lsp.dev/protocol"
)

func (def DefinitionStruct) searchParamDefinition() []protocol.Location {
	content := def.Doc.Content

	paramName, isPipelineParam := utils.GetParamNameUsedAtPos(content, def.Params.Position)

	if paramName == "" {
		return []protocol.Location{}
	}

	_, visitedNodes, _ := utils.NodeAtPos(def.Doc.RootNode, def.Params.Position)
	path := GetPathFromVisitedNodes(visitedNodes, def.Doc)

	if isPipelineParam {
		if param, ok := def.Doc.PipelinesParameters[paramName]; ok {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: param.GetRange(),
				},
			}
		}
	}

	for i := len(path) - 1; i >= 0; i-- {
		name := path[i]
		exist := def.Doc.DoesCommandOrJobOrExecutorExist(name, true)
		if !exist {
			continue
		}

		if tmp, ok := def.Doc.Commands[name]; ok {
			param := tmp.Parameters[paramName]

			if param != nil {
				return []protocol.Location{
					{
						URI:   def.Params.TextDocument.URI,
						Range: param.GetRange(),
					},
				}
			}
		} else if tmp, ok := def.Doc.Jobs[name]; ok {
			param := tmp.Parameters[paramName]

			if param != nil {
				return []protocol.Location{
					{
						URI:   def.Params.TextDocument.URI,
						Range: param.GetRange(),
					},
				}
			}
		} else if tmp, ok := def.Doc.Executors[name]; ok {
			param := tmp.GetParameters()[paramName]

			if param != nil {
				return []protocol.Location{
					{
						URI:   def.Params.TextDocument.URI,
						Range: param.GetRange(),
					},
				}
			}
		}
	}

	return []protocol.Location{}
}

func (def DefinitionStruct) searchForParamDefinition(definedParams map[string]ast.Parameter) []protocol.Location {
	for _, param := range definedParams {
		if utils.PosInRange(param.GetRange(), def.Params.Position) {
			return []protocol.Location{
				{
					URI:   def.Params.TextDocument.URI,
					Range: param.GetNameRange(),
				},
			}
		}
	}

	return []protocol.Location{}
}

func (def DefinitionStruct) searchForParamValueDefinition(callName string, params map[string]ast.ParameterValue) []protocol.Location {
	for _, param := range params {
		if utils.PosInRange(param.Range, def.Params.Position) {
			if loc, err := def.getCommandOrJobParamLocation(callName, param.Name, true); err == nil {
				return loc
			}
			return []protocol.Location{}
		}
	}

	return []protocol.Location{}
}
