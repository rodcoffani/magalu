package static

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type myParams struct {
	// see https://pkg.go.dev/github.com/invopop/jsonschema
	// json:",omitempty" tag avoids marking field as required
	SomeStringFlag string `json:",omitempty" jsonschema_description:"Optional field because of omitempty"`
	// see https://pkg.go.dev/github.com/mitchellh/mapstructure
	// renaming flags: json + mapstructure MUST MATCH!
	OtherIntFlag int `json:"int-flag" mapstructure:"int-flag" jsonschema_description:"Renamed flag"`
}

type myConfigs struct {
	SomeStringConfig string
}

type myResult struct {
	SomeResultField string
}

func newStaticExample() *core.StaticExecute {
	return core.NewStaticExecute(
		"static_example",
		"34.56",
		"static first level",
		func(ctx context.Context, params myParams, configs myConfigs) (result *myResult, err error) {
			fmt.Printf("TODO: static_example first level called. parameters=%+v, configs=%+v\n", params, configs)
			if root := core.GrouperFromContext(ctx); root != nil {
				_, _ = root.VisitChildren(func(child core.Descriptor) (run bool, err error) {
					println(">>> root child: ", child.Name())
					return true, nil
				})
			}
			if auth := core.AuthFromContext(ctx); auth != nil {
				println("I have auth from context", auth)
				return &myResult{SomeResultField: "some value"}, nil
			}
			return &myResult{SomeResultField: "some value"}, nil
		},
	)
}
