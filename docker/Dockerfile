FROM debian:11

ADD git/kubo/cmd/ipfs/ipfs /bin/ipfs
ADD git/go-ds-gcs/entrypoint /entrypoint
RUN mkdir /ipfs

ENTRYPOINT /entrypoint
