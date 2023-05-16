FROM debian:11

ADD git/kubo/cmd/ipfs/ipfs /bin/kubo
ADD git/go-ds-gcs/entrypoint /entrypoint

ENTRYPOINT /entrypoint
