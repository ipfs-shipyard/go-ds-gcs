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

	gcsds "github.com/ipfs-shipyard/go-ds-gcs"
	"github.com/ipfs/kubo/plugin"
	"github.com/ipfs/kubo/repo"
	"github.com/ipfs/kubo/repo/fsrepo"
)

const (
	defaultWorkers = 100
	defaultPrefix  = "ipfs/"

	// Use at most 1GB ram for the in memory LRU data cache.
	// IPFS blocks are max 256kB, therefore bounded at 40'000 * 256kB.
	defaultCacheSize = 40000
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
			if w, ok := v.(float64); ok {
				workers = int(w)
			} else if w, ok := v.(int); ok {
				workers = w
			} else {
				return nil, fmt.Errorf("gcsds: workers not a number: %T %v", v, v)
			}
			if workers <= 0 {
				return nil, fmt.Errorf("gcsds: workers <= 0: %d", workers)
			}
		}

		var cacheSize = defaultCacheSize
		if v, ok := m["cachesize"]; ok {
			if c, ok := v.(float64); ok {
				cacheSize = int(c)
			} else if c, ok := v.(int); ok {
				cacheSize = c
			} else {
				return nil, fmt.Errorf("gcsds: cachesize not a number: %T %v", v, v)
			}
			if cacheSize <= 0 {
				return nil, fmt.Errorf("gcsds: cachesize <= 0: %d", cacheSize)
			}
		}

		log.Printf("Parsed GCS config: bucket: %s, prefix: %s, workers: %d, cachesize: %d",
			bucket, prefix, workers, cacheSize)
		return &GcsConfig{
			cfg: gcsds.Config{
				Bucket:         bucket,
				Prefix:         prefix,
				Workers:        workers,
				DataCacheItems: cacheSize,
			},
		}, nil
	}
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
	if err != nil {
		return nil, err
	}
	if err = gd.LoadMetadata(); err != nil {
		return nil, err
	}
	return gd, nil
}
