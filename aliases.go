package gateway

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// aliasCache holds the generated alias list (from the bundled aliases.json),
// loaded once per Lambda execution environment. If ALIASES_S3_URI is set the
// indexer Lambda's freshly generated aliases.json is downloaded from S3 on cold
// start, overriding the bundled copy.
var (
	aliasOnce sync.Once
	aliasMap  map[string]string
)

// errInvalidS3URI is returned when ALIASES_S3_URI is not a valid s3 path.
var errInvalidS3URI = &s3URLError{}

type s3URLError struct{}

func (e *s3URLError) Error() string { return "ALIASES_S3_URI must be s3://bucket/key or bucket/key" }

// loadAliases returns the aliases map. On first call it fetches a fresh copy
// from S3 (if ALIASES_S3_URI is set) then falls back to the bundled aliases.json.
// Env overrides (TELA_ALIASES) and built-in aliases are applied by the caller
// (lookupAlias) so they take precedence over this generated data.
func loadAliases() map[string]string {
	aliasOnce.Do(func() {
		aliasMap = map[string]string{}
		if uri := os.Getenv("ALIASES_S3_URI"); uri != "" {
			_ = fetchAliasesFromS3(uri, aliasMap)
		}
		_ = loadBundledAliases(aliasMap)
	})
	return aliasMap
}

// fetchAliasesFromS3 downloads s3://bucket/key and merges it into dst.
// Accepts either "s3://bucket/key" or "bucket/key".
func fetchAliasesFromS3(uri string, dst map[string]string) error {
	bucket, key, err := parseS3URI(uri)
	if err != nil {
		return err
	}
	cfg, err := config.LoadDefaultConfig(ctxBackground())
	if err != nil {
		return err
	}
	client := s3.NewFromConfig(cfg)
	out, err := client.GetObject(ctxBackground(), &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	var parsed map[string]string
	if err := json.NewDecoder(out.Body).Decode(&parsed); err != nil {
		return err
	}
	for k, v := range parsed {
		dst[k] = v
	}
	return nil
}

// ctxBackground returns a background context for S3 calls.
func ctxBackground() context.Context { return context.Background() }

// parseS3URI extracts bucket and key from "s3://bucket/key" or "bucket/key".
func parseS3URI(uri string) (string, string, error) {
	u := strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(u, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errInvalidS3URI
	}
	return parts[0], parts[1], nil
}

// loadBundledAliases reads aliases.json shipped inside the binary/zip. Failures
// are non-fatal (built-in aliases still work).
func loadBundledAliases(dst map[string]string) error {
	data, err := os.ReadFile("aliases.json")
	if err != nil {
		if data2, err2 := os.ReadFile("./aliases.json"); err2 != nil {
			return err
		} else {
			data = data2
		}
	}
	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	for k, v := range parsed {
		dst[k] = v
	}
	return nil
}
