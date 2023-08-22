package gcsds

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
	"strings"

	ds "github.com/ipfs/go-datastore"
)

type Metadata struct {
	Key string
	// Store object size as int64.
	// In practice, all IPFS objects are max 256kB.
	Size int64
}

type MetadataCache struct {
	cache map[string]*Metadata
}

func NewMetadataCache() *MetadataCache {
	return &MetadataCache{
		cache: make(map[string]*Metadata),
	}
}

func (md *MetadataCache) Has(key string) bool {
	_, ok := md.cache[key]
	return ok
}

func (md *MetadataCache) Get(key string) (*Metadata, error) {
	if v, ok := md.cache[key]; ok {
		return v, nil
	}
	return nil, ds.ErrNotFound
}

func (md *MetadataCache) Put(key string, size int64) {
	md.cache[key] = &Metadata{Key: key, Size: size}
}

func (md *MetadataCache) Delete(key string) {
	delete(md.cache, key)
}

func (md *MetadataCache) Size() int {
	return len(md.cache)
}

// Offset not supported, for now.
func (md *MetadataCache) Iterator(prefix string, limit int) func() *Metadata {
	values := []*Metadata{}
	count := 0
	// TODO(leffler): Iterate consistently over map, so that offset and limit work correctly.
	for k, v := range md.cache {
		if strings.HasPrefix(k, prefix) {
			values = append(values, v)
			count++
		}
		if limit > 0 && count == limit {
			break
		}
	}

	i := 0
	l := len(values)
	f := func() *Metadata {
		if i >= l {
			return nil
		}
		v := values[i]
		i++
		return v
	}
	return f
}
