---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "mgc_network_subnetpools_book_cidr Resource - terraform-provider-mgc"
subcategory: "Network"
description: |-
  Network Subnet Pools Book CIDR
---

# mgc_network_subnetpools_book_cidr (Resource)

Network Subnet Pools Book CIDR

## Example Usage

```terraform
resource "mgc_network_subnetpools_book_cidr" "book_subnetpool" {
  cidr = "172.0.0.5/32"
  subnet_pool_id   = "example-subnetpool-id"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `cidr` (String) CIDR
- `subnet_pool_id` (String) Subnet Pool ID