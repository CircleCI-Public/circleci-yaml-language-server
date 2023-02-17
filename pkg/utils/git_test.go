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
				projectUrl: "https://github.com/circleci/circleci-vscode-extension",
			},
			want: "gh/circleci/circleci-vscode-extension",
		},
		{
			name: "bitbucket",
			args: args{
				projectUrl: "https://bitbucket.org/circleci/circleci-vscode-extension",
			},
			want: "bb/circleci/circleci-vscode-extension",
		},
		{
			name: "gitlab",
			args: args{
				projectUrl: "https://gitlab.com/circleci/circleci-vscode-extension",
			},
			want: "",
		},
		{
			name: "invalid",
			args: args{
				projectUrl: "https://invalid.com/circleci/circleci-vscode-extension",
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
