package test

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
	"testing"

	ds "github.com/ipfs/go-datastore"
	gcsds "github.com/bjornleffler/go-ds-gcs"
)

func GetGCSDatastore(t *testing.T) *gcsds.GCSDatastore {
	config := gcsds.Config{
		Bucket:  "leffler-ipfs-test",
		Prefix:  "ipfs",
		Workers: 10,
		DataCacheItems: 1000,
	}
	ds, err := gcsds.NewGCSDatastore(config)
	if err != nil {
		t.Fatalf("Failed to create data store: %v", err)
	}
	return ds
}

func TestCreateDatastore(t *testing.T) {
	GetGCSDatastore(t)
}

func TestGCSPath(t *testing.T) {
	ds := GetGCSDatastore(t)
	path := ds.GCSPath("ABC123")
	expected := "ipfs/ABC123"
	if path != expected {
		t.Fatalf("Path mismatch: %v != %v", path, expected)
	}
}

func TestCheckBucket(t *testing.T) {
	ds := GetGCSDatastore(t)
	err := ds.CheckBucket()
	if err != nil {
		t.Fatalf("Failed to check bucket access. err: %v", err)
	}
}

func TestLoadMetadata(t *testing.T) {
	ds := GetGCSDatastore(t)
	err := ds.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata. err: %v", err)
	}
}

func testPut(t *testing.T, ctx context.Context, ds *gcsds.GCSDatastore, key ds.Key, value []byte) {
	err := ds.Put(ctx, key, value)
	if err != nil {
		t.Fatalf("Failed to Put. err: %v", err)
	}
}

func testDelete(t *testing.T, ctx context.Context, ds *gcsds.GCSDatastore, key ds.Key) {
	err := ds.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to Delete 1. err: %v", err)
	}
	err = ds.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed to Delete 2. err: %v", err)
	}
}

// testNegative tests Has(), GetSize() and Get(), expecting negative.
func testNegative(t *testing.T, ctx context.Context, ds *gcsds.GCSDatastore, key ds.Key) {
	present, _ := ds.Has(ctx, key)
	if present {
		t.Fatalf("Has returned true. Expected false. key: %v", key)
	}
	_, err := ds.GetSize(ctx, key)
	if err == nil {
		t.Fatalf("GetSize expected not found error.")
	}
	_, err = ds.Get(ctx, key)
	if err == nil {
		t.Fatalf("Expected Get to fail for missing object. key: %v", key)
	}
}

// testPositive tests Has(), GetSize() and Get(), expecting positive.
func testPositive(t *testing.T, ctx context.Context, ds *gcsds.GCSDatastore, key ds.Key, value []byte) {
	present, _ := ds.Has(ctx, key)
	if !present {
		t.Fatalf("Has returned false. Expected true. key: %v", key)
	}
	size, err := ds.GetSize(ctx, key)
	if err != nil {
		t.Fatalf("GetSize expected size %d. Got err: %v", len(value), err)
	}
	if size != len(value) {
		t.Fatalf("Wrong size. Got %d. Expected %d.", size, len(value))
	}
	value2, err := ds.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to Get. err: %v", err)
	}
	if !bytes.Equal(value, value2) {
		t.Fatalf("value != value2: sizes: %d %d", len(value), len(value2))
	}
}

func TestAll(t *testing.T) {
	ds := GetGCSDatastore(t)
	key := randomKey()
	size := 100 // max IPFS block size
	value := []byte(randomSeq(size))
	ctx := context.Background()
	testNegative(t, ctx, ds, key)
	testPut(t, ctx, ds, key, value)
	testPositive(t, ctx, ds, key, value)
	testDelete(t, ctx, ds, key)
	testNegative(t, ctx, ds, key)
}

func TestReload(t *testing.T) {
	ds1 := GetGCSDatastore(t)
	key := randomKey()
	size := 100 // max IPFS block size
	value := []byte(randomSeq(size))
	ctx := context.Background()
	testNegative(t, ctx, ds1, key)
	testPut(t, ctx, ds1, key, value)
	testPositive(t, ctx, ds1, key, value)
	_ = ds1.Close()

	// Reload datastore and metadata cache.
	ds2 := GetGCSDatastore(t)
	if err := ds2.LoadMetadata(); err != nil {
		t.Fatalf("Failed to load metadata. err: %v", err)
	}
	testPositive(t, ctx, ds2, key, value)
	testDelete(t, ctx, ds2, key)
	testNegative(t, ctx, ds2, key)
}
