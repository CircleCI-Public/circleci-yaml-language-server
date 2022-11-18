package parser

import (
	"reflect"
	"testing"

	"github.com/circleci/circleci-yaml-language-server/pkg/ast"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDockerImageValue(tt.args.value)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDockerImageValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
