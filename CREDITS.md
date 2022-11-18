# Credits

This project is made possible by the community surrounding it and especially the wonderful people and projects listed in this document.

## Contributors

Developer Experience team of CircleCI

## Libraries

| Project  | License name                            | License location     |
|---|-----------------------------|------|
|github.com/adrg/xdg|MIT                          |https://github.com/adrg/xdg/blob/v0.4.0/LICENSE|
|github.com/pkg/errors|BSD-2-Clause                 |https://github.com/pkg/errors/blob/v0.8.1/LICENSE|
|github.com/segmentio/asm|MIT                          |https://github.com/segmentio/asm/blob/v1.1.3/LICENSE|
|github.com/segmentio/encoding|MIT                          |https://github.com/segmentio/encoding/blob/v0.3.5/LICENSE|
|github.com/smacker/go-tree-sitter|MIT                          |https://github.com/smacker/go-tree-sitter/blob/ec55f7cfeaf4/LICENSE|
|github.com/xeipuuv/gojsonpointer|Apache-2.0                   |https://github.com/xeipuuv/gojsonpointer/blob/4e3ac2762d5f/LICENSE-APACHE-2.0.txt|
|github.com/xeipuuv/gojsonreference|Apache-2.0                   |https://github.com/xeipuuv/gojsonreference/blob/bd5ef7bd5415/LICENSE-APACHE-2.0.txt|
|github.com/xeipuuv/gojsonschema|Apache-2.0                   |https://github.com/xeipuuv/gojsonschema/blob/v1.2.0/LICENSE-APACHE-2.0.txt|
|go.lsp.dev/jsonrpc2|BSD-3-Clause                 |https://github.com/go-language-server/jsonrpc2/blob/v0.10.0/LICENSE|
|go.lsp.dev/pkg/xcontext|BSD-3-Clause                 |https://github.com/go-language-server/pkg/blob/384b27a52fb2/LICENSE|
|go.lsp.dev/protocol|BSD-3-Clause                 |https://github.com/go-language-server/protocol/blob/v0.12.0/LICENSE|
|go.lsp.dev/uri|BSD-3-Clause                 |https://github.com/go-language-server/uri/blob/v0.3.0/LICENSE|
|go.uber.org/atomic|MIT                          |https://github.com/uber-go/atomic/blob/v1.9.0/LICENSE.txt|
|go.uber.org/multierr|MIT                          |https://github.com/uber-go/multierr/blob/v1.8.0/LICENSE.txt|
|go.uber.org/zap|MIT                          |https://github.com/uber-go/zap/blob/v1.21.0/LICENSE.txt|
|golang.org/x/mod/semver|BSD-3-Clause                 |https://cs.opensource.google/go/x/mod/+/v0.5.1:LICENSE|
|golang.org/x/sys/cpu|BSD-3-Clause                 |https://cs.opensource.google/go/x/sys/+/a9b59b02:LICENSE|
|gopkg.in/yaml.v3|MIT                          |https://github.com/go-yaml/yaml/blob/496545a6307b/LICENSE|

This list was automatically generated with the following commands:

1. task licenses
2. sbom extract
3. sbom verify ./sbom.generated.json (optional)
4. `cat sbom.generated.json | jq ".[2].packages[\"./cmd/start_server\"][] | [.name, .license, .licenseLocation] | @csv" -r`
5. Convert the CSV output to markdown with converters like [this one](https://www.convertcsv.com/csv-to-markdown.htm)

Note that sbom is an internal license util that was not open-sourced yet