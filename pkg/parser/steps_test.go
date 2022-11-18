package parser

import (
	"reflect"
	"testing"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

const YamlFile = `
steps:
    - checkout
    - checkout:
        path: /path/to/repository
    - run: ls
    - run:
        name: install deps
        command: npm install
        shell: /bin/sh
        background: true
        working_directory: /home/user/project
        no_output_timeout: 20m
        when: always
    - setup_remote_docker
    - setup_remote_docker:
        docker_layer_caching: true
        version: 1.12.6
    - save_cache:
        paths:
            - /home/user/project/cache
            - /home/user/project/cache2
        key: cache-key-1
        name: cache-name-1
        when: on_success
    - restore_cache:
        key: cache-key-1
        keys:
            - cache-key-1
            - cache-key-2
        name: cache-name-1
    - store_artifacts:
        path: /home/user/project/artifacts
        destination: circleci-docs
    - store_test_results:
        path: /home/user/project/test-results
    - persist_to_workspace:
        root: /home/user/project
        paths:
            - /home/user/project/cache
    - attach_workspace:
        at: workspace1
    - add_ssh_keys:
        fingerprints:
            - "b7:35:a6:4e:9b:0d:6d:d4:78:1e:9a:97:2a:66:6b:be"
    # Install go
    - go/install:
        version: '1.17'
    # Test
    - go/test:
        race: true
        covermode: atomic
    - when:
        condition: << pipeline.parameters.release-name >>
        steps:
            - checkout
    - unless:
        condition: << pipeline.parameters.release-name >>
        steps:
            - run: echo add release-name to enable this job
`

func TestYamlDocument_parseSteps(t *testing.T) {
	rootNode := GetRootNode([]byte(YamlFile))
	stepsNode := getFirstChildOfType(rootNode, "block_sequence").Parent() // Retrieve the block_node containing the steps

	type fields struct {
		Content []byte
	}
	type args struct {
		stepsNode *sitter.Node
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []ast.Step
	}{
		{
			name:   "End to end testing for steps",
			fields: fields{[]byte(YamlFile)},
			args:   args{stepsNode},
			want: []ast.Step{
				ast.NamedStep{Name: "checkout"},
				ast.Checkout{Path: "/path/to/repository"},
				ast.Run{Command: "ls"},
				ast.Run{
					Name:             "install deps",
					Command:          "npm install",
					Shell:            "/bin/sh",
					Background:       true,
					WorkingDirectory: "/home/user/project",
					NoOutputTimeout:  "20m",
					When:             "always",
				},
				ast.NamedStep{Name: "setup_remote_docker"},
				ast.SetupRemoteDocker{
					DockerLayerCaching: true,
					Version:            "1.12.6",
				},
				ast.SaveCache{
					Paths:     []string{"/home/user/project/cache", "/home/user/project/cache2"},
					Key:       "cache-key-1",
					CacheName: "cache-name-1",
					// When: "on_success",
				},
				ast.RestoreCache{
					Key:       "cache-key-1",
					Keys:      []string{"cache-key-1", "cache-key-2"},
					CacheName: "cache-name-1",
				},
				ast.StoreArtifacts{
					Path:        "/home/user/project/artifacts",
					Destination: "circleci-docs",
				},
				ast.StoreTestResults{
					Path: "/home/user/project/test-results",
				},
				ast.PersistToWorkspace{
					Root:  "/home/user/project",
					Paths: []string{"/home/user/project/cache"},
				},
				ast.AttachWorkspace{
					At: "workspace1",
				},
				ast.AddSSHKey{
					Fingerprints: []string{"b7:35:a6:4e:9b:0d:6d:d4:78:1e:9a:97:2a:66:6b:be"},
				},
				ast.NamedStep{
					Name: "go/install",
					Parameters: map[string]ast.ParameterValue{
						"version": {Value: "1.17", Name: "version"},
					},
				},
				ast.NamedStep{
					Name: "go/test",
					Parameters: map[string]ast.ParameterValue{
						"race": {
							Value: true,
							Name:  "race",
						},
						"covermode": {
							Value: "atomic",
							Name:  "covermode",
						},
					},
				},
				ast.NamedStep{Name: "checkout"},
				ast.Run{
					Command: "echo add release-name to enable this job",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content: tt.fields.Content,
			}
			parsedSteps := doc.parseSteps(tt.args.stepsNode)
			if len(parsedSteps) != len(tt.want) {
				t.Errorf("Parsed %v steps, expected %v", parsedSteps, tt.want)
			}

			for i := range tt.want {
				step, parsedStep := tt.want[i], parsedSteps[i]

				if reflect.TypeOf(step) != reflect.TypeOf(parsedStep) {
					t.Errorf("Parsed step %v is of type %v, expected %v", parsedStep, reflect.TypeOf(parsedStep), reflect.TypeOf(step))
				}

			}
		})
	}
}
