package sdk

import (
	"core"
	"os"
	"path/filepath"
)

// Re-exports from Core
type Descriptor = core.Descriptor
type DescriptorVisitor = core.DescriptorVisitor
type Example = core.Example
type Executor = core.Executor
type Grouper = core.Grouper
type Schema = core.Schema
type Value = core.Value

type Sdk struct {
	group *core.MergeGroup
}

func NewSdk() *Sdk {
	return &Sdk{}
}

func (o *Sdk) newOpenApiSource() *OpenApiSource {
	// TODO: are these going to be fixed? configurable?
	extensionPrefix := "x-cli"
	openApiDir := os.Getenv("MGC_SDK_OPENAPI_DIR")
	if openApiDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			openApiDir = filepath.Join(cwd, "openapis")
		}
	}

	return &OpenApiSource{
		Dir:             openApiDir,
		ExtensionPrefix: &extensionPrefix,
	}
}

func (o *Sdk) Group() core.Grouper {
	if o.group == nil {
		o.group = core.NewMergeGroup(
			"MagaLu Cloud",
			"1.0",
			"All MagaLu Groups & Executors",
			[]core.Grouper{
				newStaticRoot(),
				o.newOpenApiSource(),
			},
		)
	}
	return o.group
}