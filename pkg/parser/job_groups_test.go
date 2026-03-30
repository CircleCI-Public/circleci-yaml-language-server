package parser

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestYamlDocument_parseJobGroups(t *testing.T) {
	const singleGroup = `
    deploy-and-release:
        jobs:
          - deploy
          - release:
              requires:
                - deploy`

	const multipleGroups = `
    build-group:
        jobs:
          - build
    deploy-group:
        jobs:
          - deploy`

	const emptyGroup = `
    empty-group:
        jobs:`

	const groupNoJobsKey = `
    bare-group:
        something: else`

	type fields struct {
		Content   []byte
		JobGroups map[string]ast.JobGroup
	}
	type args struct {
		jobGroupsNode *sitter.Node
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantNames      []string
		wantJobCounts  map[string]int
		wantDAGEntries map[string]map[string][]string // group -> job -> requires
	}{
		{
			name:      "Single group with requires",
			fields:    fields{Content: []byte(singleGroup), JobGroups: make(map[string]ast.JobGroup)},
			args:      args{jobGroupsNode: getFirstChildOfType(GetRootNode([]byte(singleGroup)), "block_node")},
			wantNames: []string{"deploy-and-release"},
			wantJobCounts: map[string]int{
				"deploy-and-release": 2,
			},
			wantDAGEntries: map[string]map[string][]string{
				"deploy-and-release": {
					"deploy": {"release"},
				},
			},
		},
		{
			name:      "Multiple groups",
			fields:    fields{Content: []byte(multipleGroups), JobGroups: make(map[string]ast.JobGroup)},
			args:      args{jobGroupsNode: getFirstChildOfType(GetRootNode([]byte(multipleGroups)), "block_node")},
			wantNames: []string{"build-group", "deploy-group"},
			wantJobCounts: map[string]int{
				"build-group":  1,
				"deploy-group": 1,
			},
		},
		{
			name:      "Empty group (no jobs listed)",
			fields:    fields{Content: []byte(emptyGroup), JobGroups: make(map[string]ast.JobGroup)},
			args:      args{jobGroupsNode: getFirstChildOfType(GetRootNode([]byte(emptyGroup)), "block_node")},
			wantNames: []string{"empty-group"},
			wantJobCounts: map[string]int{
				"empty-group": 0,
			},
		},
		{
			name:      "Group without jobs key",
			fields:    fields{Content: []byte(groupNoJobsKey), JobGroups: make(map[string]ast.JobGroup)},
			args:      args{jobGroupsNode: getFirstChildOfType(GetRootNode([]byte(groupNoJobsKey)), "block_node")},
			wantNames: []string{"bare-group"},
			wantJobCounts: map[string]int{
				"bare-group": 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content:   tt.fields.Content,
				JobGroups: tt.fields.JobGroups,
			}
			doc.parseJobGroups(tt.args.jobGroupsNode)

			for _, name := range tt.wantNames {
				group, ok := doc.JobGroups[name]
				if !ok {
					t.Errorf("parseJobGroups() did not parse group %q", name)
					continue
				}
				if group.Name != name {
					t.Errorf("parseJobGroups() group.Name = %q, want %q", group.Name, name)
				}
				if wantCount, ok := tt.wantJobCounts[name]; ok {
					if len(group.JobInvocations) != wantCount {
						t.Errorf("parseJobGroups() group %q has %d job invocations, want %d", name, len(group.JobInvocations), wantCount)
					}
				}
				if wantDAG, ok := tt.wantDAGEntries[name]; ok {
					for job, wantReqs := range wantDAG {
						gotReqs, exists := group.JobsDAG[job]
						if !exists {
							t.Errorf("parseJobGroups() group %q DAG missing job %q", name, job)
							continue
						}
						if len(gotReqs) != len(wantReqs) {
							t.Errorf("parseJobGroups() group %q DAG[%q] = %v, want %v", name, job, gotReqs, wantReqs)
							continue
						}
						for i, req := range wantReqs {
							if gotReqs[i] != req {
								t.Errorf("parseJobGroups() group %q DAG[%q][%d] = %q, want %q", name, job, i, gotReqs[i], req)
							}
						}
					}
				}
			}

			if len(doc.JobGroups) != len(tt.wantNames) {
				t.Errorf("parseJobGroups() parsed %d groups, want %d", len(doc.JobGroups), len(tt.wantNames))
			}
		})
	}
}

func TestYamlDocument_parseSingleJobGroup(t *testing.T) {
	const group1 = `
    deploy-and-release:
        jobs:
          - deploy
          - release:
              requires:
                - deploy`

	type fields struct {
		Content []byte
	}
	type args struct {
		jobGroupNode *sitter.Node
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantName         string
		wantJobCount     int
		wantJobNames     []string
		wantNonZeroRange bool
	}{
		{
			name:   "Parse single job group with jobs and requires",
			fields: fields{Content: []byte(group1)},
			args: args{
				jobGroupNode: getFirstChildOfType(
					getFirstChildOfType(GetRootNode([]byte(group1)), "block_node"),
					"block_mapping_pair",
				),
			},
			wantName:         "deploy-and-release",
			wantJobCount:     2,
			wantJobNames:     []string{"deploy", "release"},
			wantNonZeroRange: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &YamlDocument{
				Content: tt.fields.Content,
			}
			got := doc.parseSingleJobGroup(tt.args.jobGroupNode)

			if got.Name != tt.wantName {
				t.Errorf("parseSingleJobGroup().Name = %q, want %q", got.Name, tt.wantName)
			}
			if len(got.JobInvocations) != tt.wantJobCount {
				t.Errorf("parseSingleJobGroup() has %d invocations, want %d", len(got.JobInvocations), tt.wantJobCount)
			}
			for i, wantName := range tt.wantJobNames {
				if i >= len(got.JobInvocations) {
					break
				}
				if got.JobInvocations[i].JobName != wantName {
					t.Errorf("parseSingleJobGroup().JobInvocations[%d].JobName = %q, want %q", i, got.JobInvocations[i].JobName, wantName)
				}
			}
			if tt.wantNonZeroRange {
				if got.Range.Start.Line == 0 && got.Range.Start.Character == 0 && got.Range.End.Line == 0 && got.Range.End.Character == 0 {
					t.Error("parseSingleJobGroup().Range is zero, expected non-zero")
				}
				if got.NameRange.Start.Line == 0 && got.NameRange.Start.Character == 0 && got.NameRange.End.Line == 0 && got.NameRange.End.Character == 0 {
					t.Error("parseSingleJobGroup().NameRange is zero, expected non-zero")
				}
				if got.JobsRange.Start.Line == 0 && got.JobsRange.Start.Character == 0 && got.JobsRange.End.Line == 0 && got.JobsRange.End.Character == 0 {
					t.Error("parseSingleJobGroup().JobsRange is zero, expected non-zero")
				}
			}
		})
	}
}

func TestYamlDocument_DoesJobGroupExist(t *testing.T) {
	doc := &YamlDocument{
		JobGroups: map[string]ast.JobGroup{
			"deploy-group": {Name: "deploy-group"},
		},
	}

	if !doc.DoesJobGroupExist("deploy-group") {
		t.Error("DoesJobGroupExist() = false for existing group")
	}
	if doc.DoesJobGroupExist("nonexistent") {
		t.Error("DoesJobGroupExist() = true for nonexistent group")
	}
}

func TestYamlDocument_FindJobGroupContainingJob(t *testing.T) {
	doc := &YamlDocument{
		JobGroups: map[string]ast.JobGroup{
			"deploy-group": {
				Name: "deploy-group",
				JobInvocations: []ast.JobInvocation{
					{JobName: "deploy"},
					{JobName: "release"},
				},
			},
			"build-group": {
				Name: "build-group",
				JobInvocations: []ast.JobInvocation{
					{JobName: "build"},
				},
			},
		},
	}

	tests := []struct {
		name      string
		jobName   string
		wantGroup string
		wantFound bool
	}{
		{"job in deploy-group", "deploy", "deploy-group", true},
		{"job in deploy-group (release)", "release", "deploy-group", true},
		{"job in build-group", "build", "build-group", true},
		{"job not in any group", "test", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGroup, gotFound := doc.FindJobGroupContainingJob(tt.jobName)
			if gotFound != tt.wantFound {
				t.Errorf("FindJobGroupContainingJob(%q) found = %v, want %v", tt.jobName, gotFound, tt.wantFound)
			}
			if gotGroup != tt.wantGroup {
				t.Errorf("FindJobGroupContainingJob(%q) group = %q, want %q", tt.jobName, gotGroup, tt.wantGroup)
			}
		})
	}
}
