package parser

import (
	"reflect"
	"testing"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
	"go.lsp.dev/protocol"
)

const YAMLFile = `
executors:
    docker-executor:
        docker:
            - image: cimg/ruby:3.0.3-browsers
              auth:
                    username: mydockerhub-user
                    password: $DOCKERHUB_PASSWORD
              environment:
                    IN_CI: true
    machine-executor:
        machine:
            image: ubuntu-2004:current
            docker_layer_caching: true
        resource_class: large
        environment:
            AWS_ECR_REGISTRY_ID: "183081753049"
    macos-executor:
        macos:
            xcode: "11.3.1"
        resource_class: large
        parameters:
            dummyParam: { type: string, default: "dummy" }
`

func TestYamlDocument_parseExecutors(t *testing.T) {
	executorsNode := getNodeForString(YAMLFile)

	type fields struct {
		Content []byte
	}
	type args struct {
		executorsNode *sitter.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]ast.Executor
	}{
		{
			name:   "End to end test for executors",
			fields: fields{[]byte(YAMLFile)},
			args:   args{executorsNode},
			want: map[string]ast.Executor{
				"docker-executor": ast.DockerExecutor{
					BaseExecutor: ast.BaseExecutor{
						Name: "docker-executor",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      2,
								Character: 4,
							},
							End: protocol.Position{
								Line:      2,
								Character: 19,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      2,
								Character: 4,
							},
							End: protocol.Position{
								Line:      9,
								Character: 31,
							},
						},
					},
					Image: []ast.DockerImage{
						{
							Image: ast.DockerImageInfo{
								Namespace: "cimg",
								Tag:       "3.0.3-browsers",
								Name:      "ruby",
								FullPath:  "cimg/ruby:3.0.3-browsers",
							},
							ImageRange: protocol.Range{
								Start: protocol.Position{
									Line:      4,
									Character: 14,
								},
								End: protocol.Position{
									Line:      4,
									Character: 45,
								},
							},
							Auth: ast.DockerImageAuth{
								Username: "mydockerhub-user",
								Password: "$DOCKERHUB_PASSWORD",
							},
							Environment: map[string]string{
								"IN_CI": "true",
							},
						},
					},
				},
				"machine-executor": ast.MachineExecutor{
					Image: "ubuntu-2004:current",
					ImageRange: protocol.Range{
						Start: protocol.Position{
							Line:      12,
							Character: 12,
						},
						End: protocol.Position{
							Line:      12,
							Character: 38,
						},
					},
					DockerLayerCaching: true,
					BaseExecutor: ast.BaseExecutor{
						Name: "machine-executor",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      10,
								Character: 4,
							},
							End: protocol.Position{
								Line:      10,
								Character: 20,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      10,
								Character: 4,
							},
							End: protocol.Position{
								Line:      16,
								Character: 47,
							},
						},
						ResourceClass: "large",
						ResourceClassRange: protocol.Range{
							Start: protocol.Position{
								Line:      14,
								Character: 8,
							},
							End: protocol.Position{
								Line:      14,
								Character: 29,
							},
						},
						BuiltInParameters: ast.ExecutableParameters{
							Environment: map[string]string{
								"AWS_ECR_REGISTRY_ID": "183081753049",
							},
						},
					},
				},
				"macos-executor": ast.MacOSExecutor{
					BaseExecutor: ast.BaseExecutor{
						Name: "macos-executor",
						NameRange: protocol.Range{
							Start: protocol.Position{
								Line:      17,
								Character: 4,
							},
							End: protocol.Position{
								Line:      17,
								Character: 18,
							},
						},
						Range: protocol.Range{
							Start: protocol.Position{
								Line:      17,
								Character: 4,
							},
							End: protocol.Position{
								Line:      23,
								Character: 0,
							},
						},
						ResourceClass: "large",
						ResourceClassRange: protocol.Range{
							Start: protocol.Position{
								Line:      20,
								Character: 8,
							},
							End: protocol.Position{
								Line:      20,
								Character: 29,
							},
						},
						UserParameters: map[string]ast.Parameter{
							"dummyParam": ast.StringParameter{
								Default: "dummy",
								BaseParameter: ast.BaseParameter{
									Name: "dummyParam",
									Range: protocol.Range{
										Start: protocol.Position{
											Line:      22,
											Character: 12,
										},
										End: protocol.Position{
											Line:      22,
											Character: 58,
										},
									},
									NameRange: protocol.Range{
										Start: protocol.Position{
											Line:      22,
											Character: 12,
										},
										End: protocol.Position{
											Line:      22,
											Character: 22,
										},
									},
									HasDefault: true,
									TypeRange: protocol.Range{
										Start: protocol.Position{
											Line:      22,
											Character: 26,
										},
										End: protocol.Position{
											Line:      22,
											Character: 38,
										},
									},
									DefaultRange: protocol.Range{
										Start: protocol.Position{
											Line:      22,
											Character: 40,
										},
										End: protocol.Position{
											Line:      22,
											Character: 56,
										},
									},
								},
							},
						},
						UserParametersRange: protocol.Range{
							Start: protocol.Position{
								Line:      21,
								Character: 8,
							},
							End: protocol.Position{
								Line:      23,
								Character: 0,
							},
						},
					},
					Xcode: "11.3.1",
					XcodeRange: protocol.Range{
						Start: protocol.Position{
							Line:      19,
							Character: 12,
						},
						End: protocol.Position{
							Line:      19,
							Character: 27,
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content:   tt.fields.Content,
				Executors: map[string]ast.Executor{},
			}

			doc.parseExecutors(tt.args.executorsNode)

			parsedExecutors := doc.Executors

			if len(parsedExecutors) != len(tt.want) {
				t.Errorf("Executors: got %v, want %v", parsedExecutors, tt.want)
				return
			}

			for i := range tt.want {
				executor, parsedExecutor := tt.want[i], parsedExecutors[i]

				if reflect.TypeOf(executor) != reflect.TypeOf(parsedExecutor) {
					t.Errorf("Executor %v is of type %v, expected %v", parsedExecutor, reflect.TypeOf(parsedExecutor), reflect.TypeOf(executor))
				}

				if !reflect.DeepEqual(executor, parsedExecutor) {
					t.Errorf("Executor got %v, want %v", parsedExecutor, executor)
				}
			}
		})
	}
}
