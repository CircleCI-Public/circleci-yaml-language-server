package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

type jobsArgs struct {
	jobsString string
}

func getJobsTests() []struct {
	name string
	args jobsArgs
	want []ast.Job
} {
	tests := []struct {
		name string
		args jobsArgs
		want []ast.Job
	}{
		{
			name: "Jobs test case 1",
			args: jobsArgs{
				jobsString: `
jobs:
    test:
        parallelism: 2
        working_directory: "~/testJob"
        shell: "superShell"
        docker:
            - circleci/node:123.4.5
            - circleci/ruby:1.2.3
        resource_class: superFast
        steps:
            - checkout`,
			},
			want: []ast.Job{
				{
					Name:             "test",
					Parallelism:      2,
					WorkingDirectory: "~/testJob",
					Shell:            "superShell",
					ResourceClass:    "superFast",
				},
			},
		},
		{
			name: "Jobs test case 2",
			args: jobsArgs{
				jobsString: `
jobs:
    test:
        steps:
            - checkout
    debug:
        parallelism: 2
        steps:
            - checkout
            - debug`,
			},
			want: []ast.Job{
				{
					Name:        "test",
					Parallelism: -1,
				},
				{
					Name:        "debug",
					Parallelism: 2,
				},
			},
		},
	}
	return tests
}

func TestYamlDocument_parseJobs(t *testing.T) {
	tests := getJobsTests()
	for _, tt := range tests {
		t.Run(tt.name+": parseJobs", func(t *testing.T) {
			doc := &YamlDocument{
				Content: []byte(tt.args.jobsString),
				Jobs:    make(map[string]ast.Job),
			}
			jobNode := getNodeForString(tt.args.jobsString)

			doc.parseJobs(jobNode)

			for _, job := range tt.want {
				if _, ok := doc.Jobs[job.Name]; !ok {
					t.Errorf("YamlDocument.parseJobs() = %s could have not been found or parsed", job.Name)
					t.Skip()
				}

				if !reflect.DeepEqual(doc.Jobs[job.Name].Name, job.Name) {
					t.Errorf("YamlDocument.parseJobs() = Name %v, want %v", doc.Jobs[job.Name], job.Name)
				}
				if !reflect.DeepEqual(doc.Jobs[job.Name].ResourceClass, job.ResourceClass) {
					t.Errorf("YamlDocument.parseJobs() = ResourceClass %v, want %v", doc.Jobs[job.Name], job.ResourceClass)
				}
				if !reflect.DeepEqual(doc.Jobs[job.Name].Shell, job.Shell) {
					t.Errorf("YamlDocument.parseJobs() = Shell %v, want %v", doc.Jobs[job.Name], job.Shell)
				}
				if !reflect.DeepEqual(doc.Jobs[job.Name].Parallelism, job.Parallelism) {
					t.Errorf("YamlDocument.parseJobs() = Parallelism %v, want %v", doc.Jobs[job.Name], job.Parallelism)
				}

				if !reflect.DeepEqual(doc.Jobs[job.Name].WorkingDirectory, job.WorkingDirectory) {
					t.Errorf("YamlDocument.parseJobs() = WorkingDirectory %v, want %v", doc.Jobs[job.Name], job.WorkingDirectory)
				}
			}
		})
	}
}

func TestYamlDocument_parseSingleJob(t *testing.T) {
	tests := getJobsTests()

	for _, tt := range tests {
		t.Run(tt.name+": parseSingleJob", func(t *testing.T) {
			rootNode := getNodeForString(tt.args.jobsString)
			doc := &YamlDocument{
				Content: []byte(tt.args.jobsString),
				Jobs:    make(map[string]ast.Job),
			}
			blockMapping := GetChildOfType(rootNode, "block_mapping")
			blockMappingPair := blockMapping.Child(0)

			job := doc.parseSingleJob(blockMappingPair)

			if !reflect.DeepEqual(tt.want[0].Name, job.Name) {
				t.Errorf("YamlDocument.parseSingleJob() = Name %v, want %v", tt.want[0], job.Name)
			}
			if !reflect.DeepEqual(tt.want[0].ResourceClass, job.ResourceClass) {
				t.Errorf("YamlDocument.parseSingleJob() = ResourceClass %v, want %v", tt.want[0], job.ResourceClass)
			}
			if !reflect.DeepEqual(tt.want[0].Shell, job.Shell) {
				t.Errorf("YamlDocument.parseSingleJob() = Shell %v, want %v", tt.want[0], job.Shell)
			}
			if !reflect.DeepEqual(tt.want[0].Parallelism, job.Parallelism) {
				t.Errorf("YamlDocument.parseSingleJob() = Parallelism %v, want %v", tt.want[0], job.Parallelism)
			}
			if !reflect.DeepEqual(tt.want[0].WorkingDirectory, job.WorkingDirectory) {
				t.Errorf("YamlDocument.parseSingleJob() = WorkingDirectory %v, want %v", tt.want[0], job.WorkingDirectory)
			}
		})
	}
}
