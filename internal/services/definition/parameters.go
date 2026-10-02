package definition

import (
	"context"

	ast2 "github.com/CircleCI-Public/circleci-yaml-language-server/internal/ast"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/paramref"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/position"
)

func (def DefinitionStruct) searchParamDefinition() []Link {
	content := def.Doc.Content

	paramName, isPipelineParam, reference := paramref.UsedAtPos(content, def.Params.Position)

	if paramName == "" {
		return []Link{}
	}

	_, visitedNodes, _ := position.NodeAt(def.Doc.RootNode, def.Params.Position)
	path := GetPathFromVisitedNodes(visitedNodes, def.Doc)

	link := func(param ast2.Parameter) []Link {
		return []Link{
			{
				Origin:    reference,
				URI:       def.Params.TextDocument.URI,
				Range:     param.GetRange(),
				NameRange: param.GetNameRange(),
			},
		}
	}

	if isPipelineParam {
		if param, ok := def.Doc.PipelineParameters[paramName]; ok {
			return link(param)
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
				return link(param)
			}
		} else if tmp, ok := def.Doc.Jobs[name]; ok {
			param := tmp.Parameters[paramName]

			if param != nil {
				return link(param)
			}
		} else if tmp, ok := def.Doc.Executors[name]; ok {
			param := tmp.GetParameters()[paramName]

			if param != nil {
				return link(param)
			}
		}
	}

	return []Link{}
}

func (def DefinitionStruct) searchForParamDefinition(definedParams map[string]ast2.Parameter) []Link {
	for _, param := range definedParams {
		if position.InRange(param.GetRange(), def.Params.Position) {
			return []Link{
				{
					URI:       def.Params.TextDocument.URI,
					Range:     param.GetNameRange(),
					NameRange: param.GetNameRange(),
				},
			}
		}
	}

	return []Link{}
}

func (def DefinitionStruct) searchForParamValueDefinition(ctx context.Context, callName string, params map[string]ast2.ParameterValue) []Link {
	for _, param := range params {
		if position.InRange(param.Range, def.Params.Position) {
			if executor := def.searchForExecutorArgument(ctx, callName, param); len(executor) > 0 {
				return executor
			}
			if loc, err := def.getCommandOrJobParamLocation(ctx, callName, param.Name, true); err == nil {
				return loc
			}
			return []Link{}
		}
	}

	return []Link{}
}
