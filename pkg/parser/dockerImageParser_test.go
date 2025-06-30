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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
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
				Digest:    "",
				FullPath:  "cimg/go:<<parameters.go_version>>",
			},
		},

		{
			name: "SHA256 digest without tag",
			args: args{
				value: "cimg/node@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "",
				Name:      "node",
				Digest:    "sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
				FullPath:  "cimg/node@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
		},

		{
			name: "SHA256 digest with tag",
			args: args{
				value: "cimg/node:22.11.0@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Tag:       "22.11.0",
				Name:      "node",
				Digest:    "sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
				FullPath:  "cimg/node:22.11.0@sha256:76aae59c6259672ab68819b8960de5ef571394681089eab2b576f85f080c73ba",
			},
		},

		{
			name: "Library image with SHA256 digest",
			args: args{
				value: "node:18@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "18",
				Name:      "node",
				Digest:    "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				FullPath:  "node:18@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},

		{
			name: "Library image with only SHA256 digest",
			args: args{
				value: "node@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Tag:       "",
				Name:      "node",
				Digest:    "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				FullPath:  "node@sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},

		{
			name: "Parse any string after @ as digest - short string",
			args: args{
				value: "cimg/go:1.24@foo",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Name:      "go",
				Tag:       "1.24",
				Digest:    "foo",
				FullPath:  "cimg/go:1.24@foo",
			},
		},

		{
			name: "Parse any string after @ as digest - no sha256 prefix",
			args: args{
				value: "cimg/node:18@abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Name:      "node",
				Tag:       "18",
				Digest:    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				FullPath:  "cimg/node:18@abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},

		{
			name: "Parse any string after @ as digest - wrong hash length",
			args: args{
				value: "cimg/go:latest@sha256:abc123",
			},
			want: ast.DockerImageInfo{
				Namespace: "cimg",
				Name:      "go",
				Tag:       "latest",
				Digest:    "sha256:abc123",
				FullPath:  "cimg/go:latest@sha256:abc123",
			},
		},

		{
			name: "Parse any string after @ as digest - non-hex characters",
			args: args{
				value: "node:alpine@sha256:ghijklmnopqrstuvwxyz1234567890abcdef1234567890abcdef1234567890",
			},
			want: ast.DockerImageInfo{
				Namespace: "library",
				Name:      "node",
				Tag:       "alpine",
				Digest:    "sha256:ghijklmnopqrstuvwxyz1234567890abcdef1234567890abcdef1234567890",
				FullPath:  "node:alpine@sha256:ghijklmnopqrstuvwxyz1234567890abcdef1234567890abcdef1234567890",
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
