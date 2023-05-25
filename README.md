# GCS Datastore Implementation for IPFS

An implementation of the [go-datastore](https://github.com/ipfs/go-datastore) interface backed by Google Cloud Storage, GCS. The implementation is based on [go-ds-s3](https://github.com/ipfs/go-ds-s3)

## Building binary
Build this plugin as an included plugin with kubo.

Clone kubo.
```bash
git clone https://github.com/ipfs/kubo
```

Get the GCS datastore plugin and add it to the preload list.
```bash
(cd kubo; go get github.com/bjornleffler/go-ds-gcs/plugin)
echo "gcsds github.com/bjornleffler/go-ds-gcs/plugin 0" >> kubo/plugin/loader/preload_list
```

Build kubo with the plugin
```bash
(cd kubo; make build)
```

(Optionally) install kubo onto the local system.
```bash
(cd kubo; make install)
```

## Building Docker container

Follow the "Building binary" section first. Clone the GCS datastore plugin and build the Docker entrypoint.
```bash
git clone https://github.com/bjornleffler/go-ds-gcs
(cd go-ds-gcs; go build docker/entrypoint.go)
```

Build docker container.
```bash
docker build -f go-ds-gcs/docker/Dockerfile -t ipfs
```

(Optional) tag and push image to repository. Replace _repository_ with your GCR repository.
```bash
docker tag ipfs gcr.io/repository/ipfs
docker push gcr.io/repository/ipfs
```

## Deployment: Docker

_ipfs_ is the image name from the previous section, "Building docker container"
```bash
docker run -d -p 4001:4001 -p 5001:5001 -p 8080:8080 --name=ipfs ipfs
```

Or you can use the tagged image from the previous section.
```bash
docker run -d -p 4001:4001 -p 5001:5001 -p 8080:8080 --name=ipfs gcr.io/repository/ipfs
```

## Deployment: Kubernetes (GKE)

1. Create a GKE cluster. The node pool must have write permission to the GCS bucket.
2. Replace _repository_ with your repository name in kubernetes/ipfs.yml
3. Replace _mynamespace_ with your namespace name in kubernetes/ipfs.yml
4. Create namespace in GKE.
```bash
kubectl create namespace mynamespace
```

5. Deploy to GKE.
```bash
kubectl apply -f kubernetes/ipfs.yml
```

6. IPFS can now be accessed at ipfs._mynamespace_ from other pods in the GKE cluster:
```bash
curl http://ipfs.mynamespace:8080/...
```

## Deployment: Local

For a brand new ipfs instance (no data stored yet):

1. Install kubo/cmd/ipfs/ipfs
2. Initialize IPFS. Replace _mybucket_ with your bucket name. Note that this depends on this [Pull Request](https://github.com/ipfs/kubo/pull/9889).
```bash
KUBO_GCS_BUCKET=mybucket ipfs init --profile gcsds
```
4. To run as a server, also run
```bash
ipfs config profile apply server
ipfs config Addresses.Gateway /ip4/0.0.0.0/tcp/8080
ipfs daemon
```

## Configuration

The config file should include the following. Replace _mybucket_ with your bucket name.
```json
{
	"Datastore": {
		"Spec": {
			"mounts": [{
				"child": {
					"bucket": "mybucket",
					"cachesize": 40000,
					"prefix": "ipfs",
					"type": "gcsds",
					"workers": 100
				},
				"mountpoint": "/blocks",
				"prefix": "flatfs.datastore",
				"type": "measure"
			}]
		}
	}
}
```

## Google Cloud credentials

Google Cloud credentials should automatically be provided when running in Google Compute Engine (GCE) or Google Kubernetes Engine (GKE). Note that for both GCE and GKE, the (node) VM needs to have write permission (scope) to GCS. For GKE, this is achieved by creating the node pool  with the "Storage read/write" [scope](https://cloud.google.com/kubernetes-engine/docs/how-to/access-scopes), which is "devstorage.read_write".

In other environments, you may have to provide credentials. One way is to use the GOOGLE_APPLICATION_CREDENTIALS environment variable. See [this document](https://cloud.google.com/docs/authentication/application-default-credentials) for more details. 

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/bjornleffler/go-ds-gcs/issues/new/choose)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md) and the Google [Code of Conduct](https://github.com/bjornleffler/go-ds-gcs/blob/master/docs/code-of-conduct.md)

## License

Apache 2.0
