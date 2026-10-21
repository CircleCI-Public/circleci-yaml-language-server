package template

import "testing"

var benchmarkTexts = []string{
	"no tags at all",
	"echo << parameters.name >> and << pipeline.git.branch >>",
	`<< pipeline.git.branch == "main" and pipeline.number > 10 >>`,
	"<<# parameters.debug >>set -x\n<</ parameters.debug >>run << parameters.cmd >>",
	`cat \<<EOF`,
}

func BenchmarkCheck(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, text := range benchmarkTexts {
			_, _ = Check(text)
		}
	}
}

func BenchmarkReferences(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		for _, text := range benchmarkTexts {
			References(text)
		}
	}
}
