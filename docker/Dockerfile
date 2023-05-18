FROM debian:11

# Install CA certs for GCS https connections to work.
RUN apt update && apt install --yes ca-certificates

ADD git/kubo/cmd/ipfs/ipfs /bin/ipfs
ADD git/go-ds-gcs/entrypoint /entrypoint

RUN mkdir /ipfs

ENTRYPOINT /entrypoint
