package parser

import (
	"reflect"
	"testing"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	"go.lsp.dev/protocol"
)

type parameterArgs struct {
	paramString string
}

func TestYamlDocument_parseParameters(t *testing.T) {
	tests := []struct {
		name    string
		args    parameterArgs
		wantRes map[string]ast.Parameter
	}{
		{
			name: "Parameters test case 1",
			args: parameterArgs{
				paramString: `parameters:
    string:
        default: "no"
        type: string
        description: "A string parameter"
    int:
        default: 1
        type: integer
        description: "An integer parameter"
    bool:
        default: true
        type: boolean
        description: "A boolean parameter"
    enumBlock:
        default: "a"
        type: enum
        description: "An enum parameter"
        enum:
            - a
            - b
            - c
    enumFlow:
        default: "a"
        type: enum
        description: "An enum parameter"
        enum: [a, b, c]
    executor:
        type: executor
        description: "An executor parameter"
    envVar:
        type: env_var_name
        default: HOME
        description: "An envVar parameter"
    steps: { type: steps }`,
			},
			wantRes: map[string]ast.Parameter{
				"string": ast.StringParameter{
					Default: "no",
					BaseParameter: ast.BaseParameter{
						Name: "string",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      1,
								Character: 4,
							},
							End: protocol.Position{
								Line:      1,
								Character: 10,
							},
						},
						Description: "A string parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      1,
								Character: 4,
							},
							End: protocol.Position{
								Line:      4,
								Character: 41,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      3,
								Character: 8,
							},
							End: protocol.Position{
								Line:      3,
								Character: 20,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      2,
								Character: 8,
							},
							End: protocol.Position{
								Line:      2,
								Character: 21,
							},
						},
					},
				},
				"int": ast.IntegerParameter{
					Default: 1,
					BaseParameter: ast.BaseParameter{
						Name: "int",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      5,
								Character: 4,
							},
							End: protocol.Position{
								Line:      5,
								Character: 7,
							},
						},
						Description: "An integer parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      5,
								Character: 4,
							},
							End: protocol.Position{
								Line:      8,
								Character: 43,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      7,
								Character: 8,
							},
							End: protocol.Position{
								Line:      7,
								Character: 21,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      6,
								Character: 8,
							},
							End: protocol.Position{
								Line:      6,
								Character: 18,
							},
						},
					},
				},
				"bool": ast.BooleanParameter{
					Default: true,
					BaseParameter: ast.BaseParameter{
						Name: "bool",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      9,
								Character: 4,
							},
							End: protocol.Position{
								Line:      9,
								Character: 8,
							},
						},
						Description: "A boolean parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      9,
								Character: 4,
							},
							End: protocol.Position{
								Line:      12,
								Character: 42,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      11,
								Character: 8,
							},
							End: protocol.Position{
								Line:      11,
								Character: 21,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      10,
								Character: 8,
							},
							End: protocol.Position{
								Line:      10,
								Character: 21,
							},
						},
					},
				},

				"enumBlock": ast.EnumParameter{
					Default: "a",
					Enum:    []string{"a", "b", "c"},
					BaseParameter: ast.BaseParameter{
						Name: "enumBlock",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      13,
								Character: 4,
							},
							End: protocol.Position{
								Line:      13,
								Character: 13,
							},
						},
						Description: "An enum parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      13,
								Character: 4,
							},
							End: protocol.Position{
								Line:      20,
								Character: 15,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      15,
								Character: 8,
							},
							End: protocol.Position{
								Line:      15,
								Character: 18,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      14,
								Character: 8,
							},
							End: protocol.Position{
								Line:      14,
								Character: 20,
							},
						},
					},
				},

				"enumFlow": ast.EnumParameter{
					Enum:    []string{"a", "b", "c"},
					Default: "a",
					BaseParameter: ast.BaseParameter{
						Name: "enumFlow",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      21,
								Character: 4,
							},
							End: protocol.Position{
								Line:      21,
								Character: 12,
							},
						},
						Description: "An enum parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      21,
								Character: 4,
							},
							End: protocol.Position{
								Line:      25,
								Character: 23,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      23,
								Character: 8,
							},
							End: protocol.Position{
								Line:      23,
								Character: 18,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      22,
								Character: 8,
							},
							End: protocol.Position{
								Line:      22,
								Character: 20,
							},
						},
					},
				},

				"executor": ast.ExecutorParameter{
					BaseParameter: ast.BaseParameter{
						Name: "executor",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      26,
								Character: 4,
							},
							End: protocol.Position{
								Line:      26,
								Character: 12,
							},
						},
						Description: "An executor parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      26,
								Character: 4,
							},
							End: protocol.Position{
								Line:      28,
								Character: 44,
							},
						},
						HasDefault: false,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      27,
								Character: 8,
							},
							End: protocol.Position{
								Line:      27,
								Character: 22,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				},

				"envVar": ast.EnvVariableParameter{
					Default: "HOME",
					BaseParameter: ast.BaseParameter{
						Name: "envVar",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      29,
								Character: 4,
							},
							End: protocol.Position{
								Line:      29,
								Character: 10,
							},
						},
						Description: "An envVar parameter",
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      29,
								Character: 4,
							},
							End: protocol.Position{
								Line:      32,
								Character: 42,
							},
						},
						HasDefault: true,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      30,
								Character: 8,
							},
							End: protocol.Position{
								Line:      30,
								Character: 26,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      31,
								Character: 8,
							},
							End: protocol.Position{
								Line:      31,
								Character: 21,
							},
						},
					},
				},
				"steps": ast.StepsParameter{
					BaseParameter: ast.BaseParameter{
						Name: "steps",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      33,
								Character: 4,
							},
							End: protocol.Position{
								Line:      33,
								Character: 9,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      33,
								Character: 4,
							},
							End: protocol.Position{
								Line:      33,
								Character: 26,
							},
						},
						HasDefault: false,
						TypeRange: protocol.Range{
							Start: protocol.Position{
								Line:      33,
								Character: 13,
							},
							End: protocol.Position{
								Line:      33,
								Character: 24,
							},
						},
						DefaultRange: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 0,
							},
							End: protocol.Position{
								Line:      0,
								Character: 0,
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramNode := getNodeForString(tt.args.paramString)
			doc := &YamlDocument{
				Content: []byte(tt.args.paramString),
			}

			if gotRes := doc.parseParameters(paramNode); !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("YamlDocument.parseParameters() = got %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestYamlDocument_parseParameterValue(t *testing.T) {
	tests := []struct {
		name    string
		args    parameterArgs
		wantRes ast.ParameterValue
		wantErr bool
	}{
		{
			name: "Simple string",
			args: parameterArgs{
				paramString: `boo: "foo bar"`,
			},
			wantErr: false,
			wantRes: ast.ParameterValue{
				Name:  "boo",
				Value: "foo bar",
				Type:  "string",
				ValueRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 5,
					},
					End: protocol.Position{
						Line:      0,
						Character: 14,
					},
				},
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: protocol.Position{
						Line:      0,
						Character: 14,
					},
				},
			},
		},
		{
			name: "Simple string without quotes",
			args: parameterArgs{
				paramString: `boo: hello`,
			},
			wantErr: false,
			wantRes: ast.ParameterValue{
				Name:  "boo",
				Value: "hello",
				Type:  "string",
				ValueRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 5,
					},
					End: protocol.Position{
						Line:      0,
						Character: 10,
					},
				},
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: protocol.Position{
						Line:      0,
						Character: 10,
					},
				},
			},
		},
		{
			name: "Simple enum",
			args: parameterArgs{
				paramString: `boo: [1, 2]`,
			},
			wantErr: false,
			wantRes: ast.ParameterValue{
				Name: "boo",
				Value: []ast.ParameterValue{
					{
						Name:  "boo",
						Type:  "integer",
						Value: 1,
						ValueRange: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 6,
							},
							End: protocol.Position{
								Line:      0,
								Character: 7,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 6,
							},
							End: protocol.Position{
								Line:      0,
								Character: 7,
							},
						},
					},
					{
						Name:  "boo",
						Type:  "integer",
						Value: 2,
						ValueRange: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 9,
							},
							End: protocol.Position{
								Line:      0,
								Character: 10,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      0,
								Character: 9,
							},
							End: protocol.Position{
								Line:      0,
								Character: 10,
							},
						},
					},
				},
				Type: "enum",
				ValueRange: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 5,
					},
					End: protocol.Position{
						Line:      0,
						Character: 11,
					},
				},
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: protocol.Position{
						Line:      0,
						Character: 11,
					},
				},
			},
		},
		{
			name: "Simple enum",
			args: parameterArgs{
				paramString: `boo:
    - 1
    - 2`,
			},
			wantErr: false,
			wantRes: ast.ParameterValue{
				Name: "boo",
				Value: []ast.ParameterValue{
					{
						Name:  "boo",
						Type:  "integer",
						Value: 1,
						ValueRange: protocol.Range{
							Start: protocol.Position{
								Line:      1,
								Character: 6,
							},
							End: protocol.Position{
								Line:      1,
								Character: 7,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      1,
								Character: 6,
							},
							End: protocol.Position{
								Line:      1,
								Character: 7,
							},
						},
					},
					{
						Name:  "boo",
						Type:  "integer",
						Value: 2,
						ValueRange: protocol.Range{
							Start: protocol.Position{
								Line:      2,
								Character: 6,
							},
							End: protocol.Position{
								Line:      2,
								Character: 7,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      2,
								Character: 6,
							},
							End: protocol.Position{
								Line:      2,
								Character: 7,
							},
						},
					},
				},
				Type: "enum",
				ValueRange: protocol.Range{
					Start: protocol.Position{
						Line:      1,
						Character: 4,
					},
					End: protocol.Position{
						Line:      2,
						Character: 7,
					},
				},
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      0,
						Character: 0,
					},
					End: protocol.Position{
						Line:      2,
						Character: 7,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paramNode := getNodeForString(tt.args.paramString).Parent()
			doc := &YamlDocument{
				Content: []byte(tt.args.paramString),
			}
			got, err := doc.parseParameterValue(paramNode)
			if (err != nil) != tt.wantErr {
				t.Errorf("YamlDocument.parseParameterValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantRes) {
				t.Errorf("YamlDocument.parseParameterValue() = got %v, want %v", got, tt.wantRes)
			}
		})
	}
}
