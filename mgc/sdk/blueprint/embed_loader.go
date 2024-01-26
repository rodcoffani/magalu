// Code generated by blueprint_index_gen. DO NOT EDIT.

//go:build embed

//nolint

package blueprint

import (
	"os"
	"syscall"
	"magalu.cloud/core/dataloader"
)

type embedLoader map[string][]byte

func GetEmbedLoader() dataloader.Loader {
	return embedLoaderInstance
}

func (f embedLoader) Load(name string) ([]byte, error) {
	if data, ok := embedLoaderInstance[name]; ok {
		return data, nil
	}
	return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
}

func (f embedLoader) String() string {
	return "embedLoader"
}

var embedLoaderInstance = embedLoader{
	"index.blueprint.yaml": ([]byte)("{\"modules\":[{\"description\":\"Operations for Block Storage API\",\"name\":\"block-storage\",\"path\":\"block-storage.blueprint.yaml\",\"url\":\"https://block-storage.magalu.cloud\",\"version\":\"1.52.0\"},{\"description\":\"Operations for Network API\",\"name\":\"network\",\"path\":\"network.blueprint.yaml\",\"url\":\"https://network.magalu.cloud\",\"version\":\"1.99.5\"}],\"version\":\"1.0.0\"}"),
	"block-storage.blueprint.yaml": ([]byte)("{\"blueprint\":\"1.0.0\",\"children\":[{\"children\":[{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Attach a volume to a virtual machine instance\",\"name\":\"create\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.write\"],\"steps\":[{\"check\":{\"errorMessageTemplate\":\"Virtual machine {{ .parameters.virtual_machine_id }} was not attached to block storage {{ .parameters.block_storage_id }}\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"attachment\\\") && $.current.result.attachment[?(@.machine_id == $.parameters.virtual_machine_id)]\\n\"},\"parameters\":{\"id\":\"$.parameters.block_storage_id\",\"virtual_machine_id\":\"$.parameters.virtual_machine_id\"},\"target\":\"/block-storage/volumes/attach\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"description\":\"Check if a volume is attached to a virtual machine instance\",\"name\":\"get\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"result\":\"{\\n  \\\"block_storage_id\\\": $.parameters.block_storage_id,\\n  \\\"virtual_machine_id\\\": $.parameters.virtual_machine_id\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.read\"],\"steps\":[{\"check\":{\"errorMessageTemplate\":\"Unable to find virtual machine {{ .parameters.virtual_machine_id }} in block storage {{ .parameters.block_storage_id }} attachments\\n\",\"jsonPathQuery\":\"hasKey($.current.result, \\\"attachment\\\") && $.current.result.attachment[?(@.machine_id == $.parameters.virtual_machine_id)]\\n\"},\"parameters\":{\"id\":\"$.parameters.block_storage_id\"},\"target\":\"/block-storage/volumes/get\"}]},{\"configsSchema\":{\"$ref\":\"/block-storage/volume-attachment/get/configsSchema\"},\"description\":\"Update a block storage volume attachment\",\"name\":\"update\",\"parametersSchema\":{\"$ref\":\"/block-storage/volume-attachment/get/parametersSchema\"},\"resultSchema\":{\"$ref\":\"/block-storage/volume-attachment/get/resultSchema\"},\"steps\":[{\"target\":\"/block-storage/volume-attachment/get\"}]},{\"configsSchema\":{\"$ref\":\"blueprint#/components/configsSchemas/default\"},\"confirm\":\"Deleting {{ .parameters.block_storage_id }} from {{ .parameters.virtual_machine_id }} cannot be undone.\\nConfirm?\\n\",\"description\":\"Detach a volume from a virtual machine instance\",\"name\":\"delete\",\"parametersSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"result\":\"{\\n  \\\"block_storage_id\\\": $.parameters.block_storage_id,\\n  \\\"virtual_machine_id\\\": $.parameters.virtual_machine_id\\n}\\n\",\"resultSchema\":{\"$ref\":\"blueprint#/components/schemas/VolumeAttachObject\"},\"scopes\":[\"block-storage.write\"],\"steps\":[{\"check\":{\"errorMessageTemplate\":\"Virtual machine {{ .parameters.virtual_machine_id }} is still attached to block storage {{ .parameters.block_storage_id }}\\n\",\"jsonPathQuery\":\"!hasKey($.current.result, \\\"attachment\\\") || $.current.result.attachment[?(@.machine_id != $.parameters.virtual_machine_id)]\\n\"},\"parameters\":{\"id\":\"$.parameters.block_storage_id\",\"virtual_machine_id\":\"$.parameters.virtual_machine_id\"},\"target\":\"/block-storage/volumes/detach\"}]}],\"description\":\"Block Storage Volume Attachment\",\"name\":\"volume-attachment\"}],\"components\":{\"configsSchemas\":{\"default\":{\"$ref\":\"/block-storage/volumes/get/configsSchema\"}},\"schemas\":{\"VolumeAttachObject\":{\"properties\":{\"block_storage_id\":{\"$ref\":\"/block-storage/volumes/attach/parametersSchema/properties/id\"},\"virtual_machine_id\":{\"$ref\":\"/block-storage/volumes/attach/parametersSchema/properties/virtual_machine_id\"}},\"type\":\"object\"}}},\"description\":\"Operations for Block Storage API\",\"name\":\"block-storage\",\"url\":\"https://block-storage.magalu.cloud\",\"version\":\"1.52.0\"}"),
	"network.blueprint.yaml": ([]byte)("{\"blueprint\":\"1.0.0\",\"children\":[{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/vpc/public-ips/create\",\"isInternal\":false}],\"description\":\"VPC Public IPs\",\"name\":\"public_ip\"},{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/security_group/rules/create\",\"isInternal\":false},{\"$ref\":\"http://magalu.cloud/sdk#/network/security_group/rules/list\",\"isInternal\":false}],\"description\":\"VPC Rules\",\"name\":\"rule\"},{\"children\":[{\"$ref\":\"http://magalu.cloud/sdk#/network/vpc/subnets/create\",\"isInternal\":false},{\"$ref\":\"http://magalu.cloud/sdk#/network/vpc/subnets/list\",\"isInternal\":false}],\"description\":\"VPC Subnets\",\"name\":\"subnets\"}],\"description\":\"Operations for Network API\",\"name\":\"network\",\"url\":\"https://network.magalu.cloud\",\"version\":\"1.99.5\"}"),
}
