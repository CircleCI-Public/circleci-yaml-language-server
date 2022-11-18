package utils

import "testing"

func TestGetYAMLBooleanValue(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true",
			args: args{
				str: "true",
			},
			want: true,
		},
		{
			name: "yes",
			args: args{
				str: "yes",
			},
			want: true,
		},
		{
			name: "y",
			args: args{
				str: "y",
			},
			want: true,
		},
		{
			name: "1",
			args: args{
				str: "1",
			},
			want: true,
		},
		{
			name: "on",
			args: args{
				str: "on",
			},
			want: true,
		},
		{
			name: "false",
			args: args{
				str: "false",
			},
			want: false,
		},
		{
			name: "no",
			args: args{
				str: "no",
			},
			want: false,
		},
		{
			name: "n",
			args: args{
				str: "n",
			},
			want: false,
		},
		{
			name: "0",
			args: args{
				str: "0",
			},
			want: false,
		},
		{
			name: "off",
			args: args{
				str: "off",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetYAMLBooleanValue(tt.args.str); got != tt.want {
				t.Errorf("GetYAMLBooleanValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
