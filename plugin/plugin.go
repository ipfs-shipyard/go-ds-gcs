package plugin

// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"fmt"
	"log"

	gcsds "github.com/ipfs/go-ds-gcs"
	"github.com/ipfs/kubo/plugin"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
)

const (
	defaultWorkers = 100
	defaultPrefix  = "ipfs/"
)

var Plugins = []plugin.Plugin{
	&GCSPlugin{},
}

type GCSPlugin struct{}

func (plugin GCSPlugin) Name() string {
	return "gcs-datastore-plugin"
}

func (plugin GCSPlugin) Version() string {
	return "0.1.0"
}

func (plugin GCSPlugin) Init(env *plugin.Environment) error {
	return nil
}

func (plugin GCSPlugin) DatastoreTypeName() string {
	log.Printf("Return datastore name.\n")
	return "gcsds"
}

func (plugin GCSPlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	// Parse config here.
	log.Printf("Parse configuration.\n")
	return func(m map[string]interface{}) (fsrepo.DatastoreConfig, error) {
		bucket, ok := m["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("gcsds: no bucket specified")
		}

		// Optional.
		var prefix = defaultPrefix
		if v, ok := m["prefix"]; ok {
			prefix = v.(string)
		}

		var workers = defaultWorkers
		if v, ok := m["workers"]; ok {
			workersf, ok := v.(float64)
			workers = int(workersf)
			switch {
			case !ok:
				return nil, fmt.Errorf("gcsds: workers not a number")
			case workers <= 0:
				return nil, fmt.Errorf("gcsds: workers <= 0: %f", workersf)
			case float64(workers) != workersf:
				return nil, fmt.Errorf("gcsds: workers is not an integer: %f", workersf)
			}
		}

		log.Printf("Parsed config: bucket: %s, prefix: %s, workers: %d", bucket, prefix, workers)
		return &GcsConfig{
			cfg: gcsds.Config{
				Bucket:  bucket,
				Prefix:  prefix,
				Workers: workers,
			},
		}, nil
	}
	return nil
}

type GcsConfig struct {
	cfg gcsds.Config
}

func (gcsConfig *GcsConfig) DiskSpec() fsrepo.DiskSpec {
	return fsrepo.DiskSpec{
		"bucket": gcsConfig.cfg.Bucket,
		"prefix": gcsConfig.cfg.Prefix,
	}
}

func (gcsConfig *GcsConfig) Create(path string) (repo.Datastore, error) {
	log.Printf("Create() path: %s\n", path)
	gd, err := gcsds.NewGCSDatastore(gcsConfig.cfg)
	if err = gd.LoadMetadata(); err != nil {
		return nil, err
	}
	return gd, nil
}
