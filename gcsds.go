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
	"bytes"
	"context"
	"fmt"
	"log"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	lru "github.com/hashicorp/golang-lru"
	ds "github.com/ipfs/go-datastore"
	dsq "github.com/ipfs/go-datastore/query"
	"google.golang.org/api/iterator"
)

var _ ds.Datastore = (*GCSDatastore)(nil)

type Config struct {
	Bucket         string
	Prefix         string
	Workers        int
	DataCacheItems int
}

type GCSDatastore struct {
	Config
	client    *storage.Client
	mdCache   *MetadataCache
	dataCache *lru.Cache
}

func NewGCSDatastore(cfg Config) (*GCSDatastore, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create GCS client: %v\n", err)
		return nil, err
	}
	dataCache, err := lru.New(cfg.DataCacheItems)
	if err != nil {
		log.Printf("Failed to create LRU cache err: %v\n", err)
		return nil, err
	}
	gd := &GCSDatastore{
		Config:    cfg,
		client:    client,
		mdCache:   NewMetadataCache(),
		dataCache: dataCache,
	}
	if err = gd.CheckBucket(); err != nil {
		return nil, err
	}
	return gd, nil
}

// CheckBucket checks that the GCS bucket exists and is accessible.
func (gd *GCSDatastore) CheckBucket() error {
	bkt := gd.client.Bucket(gd.Config.Bucket)
	_, err := bkt.Attrs(context.Background())
	if err != nil {
		// TODO(leffler): Better explanation.
		log.Printf("Failed to get attributes for bucket %s. Missing credentials? %v", gd.Config.Bucket, err)
		return err
	}
	return nil
}

// LoadMetadata pre-loads metadata for all objects in the ipfs prefix.
func (gd *GCSDatastore) LoadMetadata() error {
	listed := 0
	start := time.Now()
	ctx := context.Background()
	query := &storage.Query{Prefix: gd.Config.Prefix}
	it := gd.client.Bucket(gd.Config.Bucket).Objects(ctx, query)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Failed to load metadata for bucket: %v err: %v",
				gd.Config.Bucket, err)
			return err
		}
		// Add to cache
		key := strings.TrimPrefix(attrs.Name, gd.Config.Prefix)
		gd.mdCache.Put(key, attrs.Size)
		listed = listed + 1
	}
	elapsed := time.Since(start)
	rate := float64(listed) / elapsed.Seconds()
	log.Printf("Loaded metadata for %d object in %.2f s (%.2f objects/s)\n",
		listed, elapsed.Seconds(), rate)
	return nil
}

func (gd *GCSDatastore) Put(ctx context.Context, k ds.Key, value []byte) error {
	key := k.String()
	// log.Printf("PUT key: %v size: %d.\n", key, len(value))
	bucket := gd.client.Bucket(gd.Config.Bucket)
	path := gd.GCSPath(key)
	w := bucket.Object(path).NewWriter(ctx)
	w.ContentType = "text/plain"
	w.Metadata = map[string]string{}
	w.Write(value)
	if err := w.Close(); err != nil {
		log.Printf("Unable to close file key: %v size: %v err: %v",
			k, len(value), err)
		return err
	}
	gd.mdCache.Put(key, int64(len(value)))
	gd.dataCache.Add(key, value)
	return nil
}

func (gd *GCSDatastore) Sync(ctx context.Context, prefix ds.Key) error {
	// log.Printf("SYNC prefix: %v\n", prefix)
	return nil
}

func (gd *GCSDatastore) Get(ctx context.Context, k ds.Key) ([]byte, error) {
	// log.Printf("GET key: %v\n", k)
	key := k.String()
	if value, ok := gd.dataCache.Get(key); ok {
		if b, ok := value.([]byte); ok {
			// log.Printf("Got value from datacache. key: %s size: %d", key, len(b))
			return b, nil
		}
		log.Printf("Failed to read cached data value. Fetching from GCS. key: %v", key)
	}
	path := gd.GCSPath(key)
	obj := gd.client.Bucket(gd.Config.Bucket).Object(path)
	_, err := obj.Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, ds.ErrNotFound
	}
	if err != nil {
		log.Printf("Problem getting file from GCS: %v\n", err)
		return nil, err
	}
	// Read file.
	r, err := obj.NewReader(ctx)
	if err != nil {
		log.Printf("Problem reading file from GCS: %v\n", err)
		return nil, err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	data := buf.Bytes()
	gd.dataCache.Add(key, data)
	return data, nil
}

func (gd *GCSDatastore) Has(ctx context.Context, k ds.Key) (exists bool, err error) {
	// log.Printf("HAS key: %v\n", k)
	return gd.mdCache.Has(k.String()), nil
}

func (gd *GCSDatastore) GetSize(ctx context.Context, k ds.Key) (size int, err error) {
	// log.Printf("GETSIZE key: %v\n", k)
	md, err := gd.mdCache.Get(k.String())
	if err != nil {
		// TODO: Handle not found error.
		return -1, err
	}
	return int(md.Size), nil
}

func (gd *GCSDatastore) Delete(ctx context.Context, k ds.Key) error {
	// log.Printf("DELETE key: %v\n", k)
	bucket := gd.client.Bucket(gd.Config.Bucket)
	key := k.String()
	path := gd.GCSPath(key)
	err := bucket.Object(path).Delete(ctx)
	// Don't error for missing objects. Double deletes are OK.
	if err != nil && err != storage.ErrObjectNotExist {
		return err
	}
	gd.mdCache.Delete(key)
	gd.dataCache.Remove(key)
	return nil
}

func (gd *GCSDatastore) Query(ctx context.Context, q dsq.Query) (dsq.Results, error) {
	if len(q.Orders) > 0 || len(q.Filters) > 0 {
		msg := "GCSDatastore: Orders and Filters not supported"
		log.Print(msg)
		return nil, fmt.Errorf(msg)
	}
	if !q.KeysOnly {
		log.Printf("GCSDatastore: Requested all values for prefix '%v'. This could be expensive.", q.Prefix)
	}

	metadata := gd.mdCache.Iterator(q.Prefix, q.Limit)
	nextValue := func() (dsq.Result, bool) {
		v := metadata()
		if v == nil {
			return dsq.Result{Error: ds.ErrNotFound}, false
		}
		// Always return size, whether it was requested or not.
		entry := dsq.Entry{Key: v.Key, Size: int(v.Size)}
		if !q.KeysOnly {
			value, err := gd.Get(ctx, ds.NewKey(v.Key))
			if err != nil {
				log.Printf("GCSDatastore: Error getting value. err: %v", err)
				return dsq.Result{Error: err}, false
			}
			entry.Value = value
		}
		return dsq.Result{Entry: entry}, true
	}

	res := dsq.ResultsFromIterator(q, dsq.Iterator{
		Close: func() error {
			return nil
		},
		Next: nextValue,
	})
	return res, nil
}

func (gd *GCSDatastore) Batch(_ context.Context) (ds.Batch, error) {
	log.Printf("BATCH.\n")
	return nil, nil
}

func (gd *GCSDatastore) Close() error {
	return nil
}

func (gd *GCSDatastore) GCSPath(key string) string {
	return path.Join(gd.Config.Prefix, key)
}
