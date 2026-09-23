package validate

import "testing"

func TestIsValidDag(t *testing.T) {
	type args struct {
		dag map[string][]string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Valid DAG",
			args: args{
				map[string][]string{
					"a": {"b", "c"},
					"b": {"d", "e"},
					"c": {"f", "g"},
				},
			},
			want: 0,
		},
		{
			name: "2 way cycle",
			args: args{
				map[string][]string{
					"a": {"b"},
					"b": {"a"},
					"c": {"d"},
				},
			},
			want: 2,
		},
		{
			name: "3 way cycle",
			args: args{
				map[string][]string{
					"a": {"b"},
					"b": {"c"},
					"c": {"a"},
				},
			},
			want: 3,
		},
		{
			name: "Complex example with cycle",
			args: args{
				map[string][]string{
					"a": {"b", "c"},
					"b": {"d", "e"},
					"c": {"e"},
					"d": {"f", "g"},
					"e": {"a"},
				},
			},
			want: 7,
		},
		{
			name: "Complex example without cycle",
			args: args{
				map[string][]string{
					"a": {"b", "c"},
					"b": {"d", "e"},
					"c": {"f", "g"},
					"d": {"f", "g"},
					"e": {"f"},
					"z": {"x"},
				},
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidDAG(tt.args.dag); len(got) != tt.want {
				t.Errorf("IsValidDag() = %v, want %v", got, tt.want)
			}
		})
	}
}
