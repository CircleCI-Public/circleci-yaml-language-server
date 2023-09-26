package utils

import "testing"

func Test_fromUrlToProjectSlug(t *testing.T) {
	type args struct {
		projectUrl string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "github",
			args: args{
				projectUrl: "https://github.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh github",
			args: args{
				projectUrl: "git@github.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "gh/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "bitbucket",
			args: args{
				projectUrl: "https://bitbucket.org/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "bb/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "ssh bitbucket",
			args: args{
				projectUrl: "git@bitbucket.org:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "bb/CircleCI-Public/circleci-yaml-language-server",
		},
		{
			name: "gitlab",
			args: args{
				projectUrl: "https://gitlab.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "",
		},
		{
			name: "ssh gitlab",
			args: args{
				projectUrl: "git@gitlab.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "",
		},
		{
			name: "invalid",
			args: args{
				projectUrl: "https://invalid.com/CircleCI-Public/circleci-yaml-language-server",
			},
			want: "",
		},
		{
			name: "ssh invalid",
			args: args{
				projectUrl: "git@invalid.com:CircleCI-Public/circleci-yaml-language-server.git",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fromUrlToProjectSlug(tt.args.projectUrl); got != tt.want {
				t.Errorf("fromUrlToProjectSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}
