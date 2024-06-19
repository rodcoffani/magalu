# Creates a new database instance asynchronously for a tenant.

## Usage:
```bash
Usage:
  ./mgc dbaas instances create [flags]
```

## Product catalog:
- Examples:
- ./mgc dbaas instances create --datastore-id="063f3994-b6c2-4c37-96c9-bab8d82d36f7" --flavor-id="8bbe8e01-40c8-4d2b-80e8-189debc44b1c" --name="dbaas-name" --password="dbaas-password" --user="dbaas-user" --volume.size=30 --volume.type="CLOUD_NVME"

## Other commands:
- Flags:
- --backup-retention-days integer   Backup Retention Days (min: 1) (default 7)
- --backup-start-at time            Backup Start At (default "04:00:00")
- --cli.list-links enum[=table]     List all available links for this command (one of "json", "table" or "yaml")
- --cli.watch                       Wait until the operation is completed by calling the 'get' link and waiting until termination. Akin to '! get -w'
- --datastore-id uuid               Datastore Id
- --engine-id uuid                  Engine Id
- --exchange string                 Exchange (default "dbaas-internal")
- --flavor-id uuid                  Flavor Id (required)
- -h, --help                            help for create
- --name string                     Name (max character count: 100) (required)
- --parameters array(object)        Parameters
- Use --parameters=help for more details (default [])
- --password string                 Password (max character count: 50) (required)
- --user string                     User (max character count: 25) (required)
- -v, --version                         version for create
- --volume object                   InstanceVolumeRequest (properties: size and type)
- Use --volume=help for more details (required)
- --volume.size integer             InstanceVolumeRequest: Size (range: 10 - 50000)
- This is the same as '--volume=size:integer'.
- --volume.type enum                InstanceVolumeRequest: An enumeration. (one of "CLOUD_HDD", "CLOUD_NVME" or "CLOUD_NVME_15K")
- This is the same as '--volume=type:enum'. (default "CLOUD_NVME_15K")

## Flags:
```bash
Global Flags:
      --cli.show-cli-globals   Show all CLI global flags on usage text
      --env enum               Environment to use (one of "pre-prod" or "prod") (default "prod")
      --region enum            Region to reach the service (one of "br-mgl1", "br-ne1" or "br-se1") (default "br-se1")
      --server-url uri         Manually specify the server to use
```
