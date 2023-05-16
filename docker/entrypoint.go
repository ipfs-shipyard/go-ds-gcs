package main

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
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
)

type Config struct {
	Bucket       string
	IpfsPath     string
	Project      string
	Prefix       string
	SleepSeconds int
}

var (
	cfg    = Config{}
	client *storage.Client
)

func parseArgs() Config {
	fs := flag.CommandLine
	fs.StringVar(&cfg.Bucket, "bucket", "", "GCS bucket name.")
	fs.StringVar(&cfg.IpfsPath, "path", "/ipfs", "IPFS disk path.")
	fs.StringVar(&cfg.Project, "project", "", "GCP project name.")
	fs.StringVar(&cfg.Prefix, "prefix", "ipfs/", "IPFS prefix in GCS bucket.")
	fs.IntVar(&cfg.SleepSeconds, "sleep", 60, "Seconds to sleep on failure.")
	flag.Parse()
	return cfg
}

func fail(msg string) {
	log.Printf(msg)
	log.Printf("Sleep %d seconds.", cfg.SleepSeconds)
	time.Sleep(time.Duration(cfg.SleepSeconds) * time.Second)
	os.Exit(1)
}

// GetProject gets the GCP project ID from GCP crecentials.
func getProject() string {
	log.Printf("Get project from GCP credentials.")
	ctx := context.Background()
	credentials, err :=
		google.FindDefaultCredentials(ctx, compute.ComputeScope)
	// TODO(leffler): Explain how to specify credentials.
	msg := "Failed to get project id. Please specify with -project option."
	if err != nil {
		msg = fmt.Sprintf("%s err: %v\n", msg, err)
		fail(msg)
	}
	project := credentials.ProjectID
	if project == "" {
		fail(msg)
	}
	return project
}

// listBuckets lists buckets in a project.
func listBuckets(projectID string) (buckets []string, err error) {
	it := client.Buckets(context.Background(), projectID)
	for {
		battrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return buckets, err
		}
		bucket := battrs.Name
		// Ignore GCR artifact buckets
		if strings.HasPrefix(bucket, "artifacts.") &&
			strings.HasSuffix(bucket, ".a.appspot.com") {
			continue
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

func foundPrefixInGCS(project, bucket, prefix string) bool {
	ctx := context.Background()
	q := storage.Query{Prefix: prefix}
	it := client.Bucket(bucket).Objects(ctx, &q)
	for {
		_, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fail(fmt.Sprintf("%v", err))
		}
		return true
	}
	return false
}

func chooseProject(cfg *Config) (project string) {
	project = cfg.Project
	if project == "" {
		project = getProject()
	}
	return project
}

// checkBucket chooses an appropriate bucket to use for IPFS.
func chooseBucket(cfg *Config, project string) (bucket string) {
	bucket = cfg.Bucket
	if bucket == "" {
		// List all available buckets in GCP project.
		buckets, _ := listBuckets(project)
		if len(buckets) == 0 {
			fail("No buckets found.")
		}
		// Pick any bucket with existing contents
		for _, b := range buckets {
			if foundPrefixInGCS(project, b, cfg.Prefix) {
				return b
			}
		}
		// Otherwise, just pick the first one.
		return buckets[0]
	}
	return bucket
}

// checkBucket checks that the bucket is writeable.
func checkBucket(project, bucket, prefix string) {
	ctx := context.Background()
	object := cfg.Prefix + "test"
	o := client.Bucket(bucket).Object(object)
	w := o.NewWriter(ctx)
	w.ContentType = "text/plain"
	if err := w.Close(); err != nil {
		// TODO(leffler): Explain how to fix this. GCE VM premission?
		fail(fmt.Sprintf("Failed to create file %v in bucket %v.", object, bucket))
	}
	// Clean up test file.
	if err := o.Delete(ctx); err != nil {
		log.Printf("Failed to delete gs://%s/%s", bucket, object)
	}
}

func configureIPFS(cfg *Config, bucket string) {
	configPath := fmt.Sprintf("%s/%s", cfg.IpfsPath, "/config")
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("IPFS already configured.")
		return
	}
	log.Printf("-------------------------------------------------")
	log.Printf("Configure IPFS for bucket %v", bucket)
	log.Printf("-------------------------------------------------")
	cmd1 := exec.Command("ipfs", "init", "--profile", "gcsds")
	cmd1.Env = append(cmd1.Environ(), fmt.Sprintf("IPFS_PATH=%s", cfg.IpfsPath))
	cmd1.Env = append(cmd1.Environ(), fmt.Sprintf("KUBO_GCS_BUCKET=%s", bucket))
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	cmd1.Run()
}

func startIPFS(cfg *Config) {
	log.Printf("-------------------------------------------------")
	log.Printf("Start IPFS")
	log.Printf("-------------------------------------------------")
	cmd2 := exec.Command("ipfs", "daemon")
	cmd2.Env = append(cmd2.Environ(), fmt.Sprintf("IPFS_PATH=%s", cfg.IpfsPath))
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	cmd2.Run()
}

func main() {
	cfg := parseArgs()

	var err error
	client, err = storage.NewClient(context.Background())
	if err != nil {
		fail(fmt.Sprintf("Failed to create GCS client: %v", err))
	}

	// 1. Choose a GCP project.
	project := chooseProject(&cfg)
	log.Printf("Using GCP project: %v", project)

	// 2. Choose a GCS bucket.
	bucket := chooseBucket(&cfg, project)
	log.Printf("Using GCS bucket: %v", bucket)

	// 3. Check that GCS Bucket is writable.
	checkBucket(project, bucket, cfg.Prefix)
	log.Printf("GCS bucket %v is writeable.", bucket)

	// 4. Configure IPFS. Once only.
	configureIPFS(&cfg, bucket)

	// 5. Start IPFS server.
	startIPFS(&cfg)
}
