# GCS Datastore Implementation for IPFS

This is an implementation of the datastore interface backed by Google Cloud Storage, GCS.

The implementation is based on [go-ds-s3](https://github.com/ipfs/go-ds-s3)

**NOTE:** Plugins only work on Linux and MacOS at the moment. You can track the progress of this issue here: https://github.com/golang/go/issues/19282

## Building 

As go plugins can be finicky to correctly compile and install, the easiest way to build this plugin is to also re-build kubo.

```bash

# Clone kubo.
> git clone https://github.com/ipfs/kubo
> cd kubo

# Pull in the datastore plugin (you can specify a version other than latest if you'd like).
> go get github.com/bjornleffler/go-ds-gcs/plugin@latest

# Add the plugin to the preload list.
> echo -en "\gcsds github.com/bjornleffler/go-ds-gcs/plugin 0" >> plugin/loader/preload_list

# ( this first pass will fail ) Try to build kubo with the plugin
> make build

# Update the deptree
> go mod tidy

# Now rebuild kubo with the plugin
> make build

# (Optionally) install kubo
> make install
```

## Detailed Installation

For a brand new ipfs instance (no data stored yet):

1. Copy kubo/cmd/ipfs/ipfs to where you want it installed.
2. Run `KUBO_GCS_BUCKET=mybucket ipfs init --profile gcsds` (depending on code yet to be submitted).
3. To run as a server, also run `ipfs config profile apply server` followed by `ipfs daemon`

### Configuration

The config file should include the following:
```json
{
  "Datastore": {
  ...

    "Spec": {
      "mounts": [
        {
          "child": {
            "bucket": "MYBUCKET",
            "cachesize": 40000,
            "prefix": "ipfs",
            "type": "gcsds",
            "workers": 100
          },
          "mountpoint": "/blocks",
          "prefix": "flatfs.datastore",
          "type": "measure"
        },
```

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/bjornleffler/go-ds-gcs/issues/new/choose)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md) and Google [Code of Conduct](https://github.com/bjornleffler/go-ds-gcs/blob/master/docs/code-of-conduct.md)

## License

Apache 2.0
