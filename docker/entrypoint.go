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
	client       *storage.Client
	SleepSeconds int
)

func parseArgs() Config {
	cfg := Config{}
	fs := flag.CommandLine
	fs.StringVar(&cfg.Bucket, "bucket", "", "GCS bucket name.")
	fs.StringVar(&cfg.IpfsPath, "path", os.Getenv("IPFS_PATH"),
		"IPFS disk path. Default: $IPFS_PATH")
	fs.StringVar(&cfg.Project, "project", "", "GCP project name.")
	fs.StringVar(&cfg.Prefix, "prefix", "ipfs/", "IPFS prefix in GCS bucket.")
	fs.IntVar(&SleepSeconds, "sleep", 10, "Seconds to sleep on failure.")
	flag.Parse()
	return cfg
}

func exit() {
	log.Printf("Sleep %d seconds to avoid aggressive retries.", SleepSeconds)
	time.Sleep(time.Duration(SleepSeconds) * time.Second)
	os.Exit(1)
}

func getStorageClient() {
	var err error
	client, err = storage.NewClient(context.Background())
	if err != nil {
		log.Printf("Failed to create GCS client: %v", err)
		exit()
	}
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
		log.Printf("%s err: %v\n", msg, err)
		exit()
	}
	project := credentials.ProjectID
	if project == "" {
		log.Print(msg)
		exit()
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
	_, err := it.Next()
	if err == iterator.Done {
		return false
	}
	if err != nil {
		log.Printf("%v", err)
		exit()
	}
	return true
}

func chooseProject(cfg *Config) {
	if cfg.Project == "" {
		cfg.Project = getProject()
	}
	if cfg.Project == "" {
		log.Printf("Failed to get project name. Please specify project with -project")
		exit()
	}
}

// checkBucket chooses an appropriate bucket to use for IPFS.
func chooseBucket(cfg *Config) {
	if cfg.Bucket == "" {
		// List all available buckets in GCP project.
		buckets, _ := listBuckets(cfg.Project)
		if len(buckets) > 0 {
			// Pick any bucket with existing contents
			for _, b := range buckets {
				if foundPrefixInGCS(cfg.Project, b, cfg.Prefix) {
					cfg.Bucket = b
					return
				}
			}
			// Otherwise, just pick the first one.
			cfg.Bucket = buckets[0]
		}
	}
	if cfg.Bucket == "" {
		log.Printf("Failed to choose a bucket. Please specify with -bucket argument.")
		exit()
	}
}

// checkBucket checks that the chosen bucket is writeable.
func checkBucket(cfg *Config) {
	ctx := context.Background()
	object := cfg.Prefix + "test"
	o := client.Bucket(cfg.Bucket).Object(object)
	w := o.NewWriter(ctx)
	w.ContentType = "text/plain"
	if err := w.Close(); err != nil {
		log.Printf("Failed to create gs://%s/%s", cfg.Bucket, object)
		log.Printf("err: %v", err)
		log.Printf("Does this system have write permissions to GCS?")
		log.Printf("For GKE, does the node pool have \"Storage read/write\" permissions? (devstorage.read_write)")
		exit()
	}
	// Clean up test file.
	if err := o.Delete(ctx); err != nil {
		log.Printf("Failed to delete gs://%s/%s. Ignore and continue.", cfg.Bucket, object)
	}
}

func run(args []string, env ...string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(cmd.Env, env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Printf("-------------------------------------------------")
	log.Printf("# Command: %v", cmd)
	log.Printf("# Environment: %v", env)
	log.Printf("-------------------------------------------------")
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func configureIPFS(cfg *Config) {
	configPath := fmt.Sprintf("%s/%s", cfg.IpfsPath, "/config")
	if _, err := os.Stat(configPath); err == nil {
		log.Printf("IPFS already configured.")
		return
	}
	log.Printf("-------------------------------------------------")
	log.Printf("Configure IPFS for bucket %v", cfg.Bucket)
	ipfsPath := fmt.Sprintf("IPFS_PATH=%s", cfg.IpfsPath)
	bucket := fmt.Sprintf("KUBO_GCS_BUCKET=%s", cfg.Bucket)
	// Bootstrap IPFS with GCS datastore.
	run([]string{"ipfs", "init", "--profile", "gcsds"}, bucket, ipfsPath)
	// Apply server profile. This disables announcements to local networks.
	run([]string{"ipfs", "config", "profile", "apply", "server"}, ipfsPath)
	// Expose gateway port.
	run([]string{"ipfs", "config", "Addresses.Gateway", "/ip4/0.0.0.0/tcp/8080"})
	// Expose admin port.
	// run([]string{"ipfs", "config", "Addresses.API", "/ip4/0.0.0.0/tcp/5001"})
}

func startIPFS(cfg *Config) {
	log.Printf("-------------------------------------------------")
	log.Printf("Start IPFS")
	ipfsPath := fmt.Sprintf("IPFS_PATH=%s", cfg.IpfsPath)
	run([]string{"ipfs", "daemon"}, ipfsPath)
}

func main() {
	cfg := parseArgs()
	getStorageClient()

	// 1. Choose a GCP project.
	chooseProject(&cfg)
	log.Printf("Using IPFS Path: '%v'", cfg.IpfsPath)
	log.Printf("Using GCP project: '%v'", cfg.Project)

	// 2. Choose a GCS bucket.
	chooseBucket(&cfg)
	log.Printf("Using GCS bucket: '%v'", cfg.Bucket)

	// 3. Check that GCS Bucket is writable.
	checkBucket(&cfg)
	log.Printf("GCS bucket %v is writeable.", cfg.Bucket)

	// 4. Configure IPFS. Once only.
	configureIPFS(&cfg)

	// 5. Start IPFS server.
	startIPFS(&cfg)
}
