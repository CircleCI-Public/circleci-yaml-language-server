package parser_test

import (
	"testing"

	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/parser"
	"github.com/CircleCI-Public/circleci-yaml-language-server/internal/testing/testHelpers"
)

func BenchmarkParseForCompile(b *testing.B) {
	for _, c := range benchConfigs {
		b.Run(c.name, func(b *testing.B) {
			content := []byte(c.config)
			b.ReportAllocs()
			for b.Loop() {
				doc, err := parser.ParseForCompile(content, "config.yml", testHelpers.DefaultSettings())
				if err != nil {
					b.Fatal(err)
				}
				doc.Close()
			}
		})
	}
}
