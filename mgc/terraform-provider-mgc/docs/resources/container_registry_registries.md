---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_container_registry_registries Resource - terraform-provider-mgc"
subcategory: ""
description: |-
  Routes related to creation, listing and deletion of registries.
---

# mgc_container_registry_registries (Resource)

Routes related to creation, listing and deletion of registries.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) A unique, global name for the container registry. It must be written in lowercase letters and consists only of numbers and letters, up to a limit of 63 characters.

### Read-Only

- `created_at` (String) Date and time of creation of the container registry.
- `id` (String) Container Registry's UUID.
- `storage_usage_bytes` (Number) Storage used in bytes.
- `updated_at` (String) Date and time of the last change to the container registry.