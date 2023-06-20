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
	"testing"

	gcsds "github.com/ipfs-shipyard/go-ds-gcs"
)

func TestEntry(t *testing.T) {
	md := gcsds.NewMetadataCache()
	key := randomKey().String()
	size := int64(1000)
	if md.Has(key) {
		t.Fatalf("Key existed too early.")
	}
	md.Put(key, size)
	if !md.Has(key) {
		t.Fatalf("Key expected.")
	}
	m, err := md.Get(key)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if m.Size != size {
		t.Fatalf("Wrong size. Expected %d got: %d", size, m.Size)
	}
	md.Delete(key)
	if md.Has(key) {
		t.Fatalf("Delete entry still exists.")
	}
}

func TestIteratorEmpty(t *testing.T) {
	md := gcsds.NewMetadataCache()
	iterator := md.Iterator("", 0)
	v := iterator()
	if v != nil {
		t.Fatalf("Expected nil value from empty metadata cache.")
	}
}

func TestIteratorSingleEntry(t *testing.T) {
	md := gcsds.NewMetadataCache()
	key := randomKey().String()
	size := int64(1000)
	md.Put(key, size)
	iterator := md.Iterator("", 0)
	v := iterator()
	if v == nil {
		t.Fatalf("Expected single value from single entry metadata cache.")
	}
	if v.Key != key {
		t.Fatalf("Expected key: %v got key: %v", key, v.Key)
	}
	if v.Size != size {
		t.Fatalf("Expected size: %v got size: %v", size, v.Size)
	}
}

func metadataCacheWithEntries() *gcsds.MetadataCache {
	md := gcsds.NewMetadataCache()
	key1 := "AA" + randomKey().String()
	key2 := "AB" + randomKey().String()
	key3 := "BB" + randomKey().String()
	size1 := int64(1001)
	size2 := int64(1002)
	size3 := int64(1003)
	md.Put(key1, size1)
	md.Put(key2, size2)
	md.Put(key3, size3)
	return md
}

func getEntries(it func() *gcsds.Metadata) map[string]int64 {
	entries := map[string]int64{}
	m := it()
	for m != nil {
		entries[m.Key] = m.Size
		m = it()
	}
	return entries
}

func TestIteratorTwoEntries(t *testing.T) {
	md := metadataCacheWithEntries()
	it := md.Iterator("", 0)
	entries := getEntries(it)
	if len(entries) != md.Size() {
		t.Fatalf("Expected %d entries. Got: %d", md.Size(), len(entries))
	}
}

// Offset not supported yet.
// func TestIteratorOffset(t *testing.T) {
//	md := metadataCacheWithEntries()
//	it := md.Iterator("", 0)
//	entries := getEntries(it)
//	expected := md.Size() - 1
//	if len(entries) != expected {
//		t.Fatalf("Expected %d entries. Got: %d", expected, len(entries))
//	}
//}

func TestIteratorPrefix(t *testing.T) {
	md := metadataCacheWithEntries()
	it := md.Iterator("A", 0)
	entries := getEntries(it)
	expected := 2
	if len(entries) != expected {
		t.Fatalf("Expected %d entries. Got: %d", expected, len(entries))
	}
}

func TestIteratorLimit(t *testing.T) {
	md := metadataCacheWithEntries()
	it := md.Iterator("", 2)
	entries := getEntries(it)
	expected := 2
	if len(entries) != expected {
		t.Fatalf("Expected %d entries. Got: %d", expected, len(entries))
	}
}
