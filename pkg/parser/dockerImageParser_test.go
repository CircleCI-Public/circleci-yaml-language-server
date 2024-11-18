package parser

import (
	"reflect"
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/pkg/ast"
)

func Test_parseDockerImageValue(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    ast.DockerImageInfo
		wantErr bool
	}{
		{
			name: "",
			args: args{
				value: "node",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "",
				Name:      "node",
				FullPath:  "node",
			},
		},

		{
			name: "",
			args: args{
				value: "node:",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "",
				Name:      "node",
				FullPath:  "node:",
			},
		},

		{
			name: "",
			args: args{
				value: "node:12",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "12",
				Name:      "node",
				FullPath:  "node:12",
			},
		},

		{
			name: "",
			args: args{
				value: "node:latest",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "latest",
				Name:      "node",
				FullPath:  "node:latest",
			},
		},

		{
			name: "",
			args: args{
				value: "cimg/go:latest",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "latest",
				Name:      "go",
				FullPath:  "cimg/go:latest",
			},
		},

		{
			name: "",
			args: args{
				value: "cimg/go",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "",
				Name:      "go",
				FullPath:  "cimg/go",
			},
		},

		{
			name: "",
			args: args{
				value: "cimg/go:",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "",
				Name:      "go",
				FullPath:  "cimg/go:",
			},
		},

		{
			name: "",
			args: args{
				value: "cimg/go:<<parameters.go_version>>",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "<<parameters.go_version>>",
				Name:      "go",
				FullPath:  "cimg/go:<<parameters.go_version>>",
			},
		},

		{
			name: "",
			args: args{
				value: "cimg/node:22.11.0@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "22.11.0",
				Name:      "node",
				FullPath:  "cimg/node:22.11.0@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDockerImageValue(tt.args.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDockerImageValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
