package config

import (
	"context"

	"slices"

	"github.com/jedib0t/go-pretty/v6/table"
	"magalu.cloud/core"
	mgcUtilsPkg "magalu.cloud/core/utils"
	"magalu.cloud/sdk/static/config/common"
)

type configInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

func configListFormatter(exec core.Executor, result core.Result) string {
	// it must be this, no need to check
	resultWithValue, _ := core.ResultAs[core.ResultWithValue](result)
	configMap := resultWithValue.Value().(map[string]any)

	writer := table.NewWriter()
	writer.AppendHeader(table.Row{"Name", "Type", "Description"})

	sortedKeys := make([]string, 0, len(configMap))
	for k := range configMap {
		sortedKeys = append(sortedKeys, k)
	}
	slices.Sort(sortedKeys)

	for _, k := range sortedKeys {
		info := configMap[k].(map[string]any)
		writer.AppendRow(table.Row{k, info["type"], info["description"]})
	}

	return writer.Render()
}

var getList = mgcUtilsPkg.NewLazyLoader[core.Executor](newList)

func newList() core.Executor {
	executor := core.NewStaticExecuteSimple(
		core.DescriptorSpec{
			Name:        "list",
			Description: "List all available Configs",
		},
		getAllConfigs,
	)
	return core.NewExecuteFormat(executor, configListFormatter)
}

func getAllConfigs(ctx context.Context) (map[string]configInfo, error) {
	configSchemas, err := common.ListAllConfigSchemas(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]configInfo, len(configSchemas))
	for name, schema := range configSchemas {
		result[name] = configInfo{
			Name:        name,
			Type:        schema.Type,
			Description: schema.Description,
		}
	}

	return result, nil
}
