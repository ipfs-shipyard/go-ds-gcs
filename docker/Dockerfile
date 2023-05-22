FROM debian:11

# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Install CA certs for GCS https connections to work.
RUN apt update && apt install --yes ca-certificates

ADD git/kubo/cmd/ipfs/ipfs /bin/ipfs
ADD git/go-ds-gcs/entrypoint /entrypoint

ENV IPFS_PATH /ipfs
RUN mkdir -p $IPFS_PATH

ENTRYPOINT /entrypoint
