# netbox-oxidized-sync
Sync Oxidized data to Netbox.

The device has to exists within netbox with the same name.  
This is because we need to get the tenant and the site from netbox. (oxidized does not have that info)

Currently supports FortiOS.  
Within FortiOS, the following items are synced

| Type  | Supported  |
|---|---|
| Vlans  | &check;  |
| Aggregate ports  | &check; |
| Virtual switches  | &check;  |
| Redundant Ports | &check;   |
| Normal Ports | &check; |
| Ip Adresses |  &cross; |


## Use
Currently there is no published binary so you have to build it yourself.  

Clone the repo and run `go build cmd/netbox-oxidized-sync/netbox-oxidized-sync.go`  
Then you can run `./netbox-oxidized-sync` to run the binary.

To configure the application copy the `configs/example.settings.json` to `configs/settings.json` and modify where needed.
