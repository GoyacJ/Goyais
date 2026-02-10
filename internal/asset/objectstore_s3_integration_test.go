package asset

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"goyais/internal/command"
)

func TestS3CompatibleStoreIntegration(t *testing.T) {
	endpoint := normalizeEndpoint(readFirstNonEmpty(
		"GOYAIS_IT_OBJECT_STORE_ENDPOINT",
		"GOYAIS_IT_MINIO_ENDPOINT",
	))
	accessKey := readFirstNonEmpty(
		"GOYAIS_IT_OBJECT_STORE_ACCESS_KEY",
		"GOYAIS_IT_MINIO_ACCESS_KEY",
	)
	secretKey := readFirstNonEmpty(
		"GOYAIS_IT_OBJECT_STORE_SECRET_KEY",
		"GOYAIS_IT_MINIO_SECRET_KEY",
	)
	bucket := readFirstNonEmpty(
		"GOYAIS_IT_OBJECT_STORE_BUCKET",
		"GOYAIS_IT_MINIO_BUCKET",
	)

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Skip("set GOYAIS_IT_OBJECT_STORE_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET to enable object store integration test")
	}

	useSSL := strings.EqualFold(strings.TrimSpace(readFirstNonEmpty(
		"GOYAIS_IT_OBJECT_STORE_USE_SSL",
		"GOYAIS_IT_MINIO_USE_SSL",
	)), "true")

	reqCtx := command.RequestContext{
		TenantID:    "t-it-store",
		WorkspaceID: "w-it-store",
		UserID:      "u-it-store",
		OwnerID:     "u-it-store",
	}
	now := time.Now().UTC()

	for _, provider := range []string{"minio", "s3"} {
		provider := provider
		t.Run(provider, func(t *testing.T) {
			t.Parallel()

			store := NewObjectStore(ObjectStoreOptions{
				Provider:  provider,
				Endpoint:  endpoint,
				AccessKey: accessKey,
				SecretKey: secretKey,
				Bucket:    bucket,
				Region:    "us-east-1",
				UseSSL:    useSSL,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			if err := store.Ping(ctx); err != nil {
				t.Fatalf("ping %s object store: %v", provider, err)
			}

			content := []byte("object-store-integration-" + provider + "-" + now.Format(time.RFC3339Nano))
			hash := provider + "-" + now.Format("20060102150405.000000000")

			uri, err := store.Put(ctx, reqCtx, hash, content, now)
			if err != nil {
				t.Fatalf("put %s object: %v", provider, err)
			}
			if uri == "" {
				t.Fatalf("expected uri for %s object", provider)
			}

			got, err := store.Get(ctx, uri)
			if err != nil {
				t.Fatalf("get %s object: %v", provider, err)
			}
			if !bytes.Equal(got, content) {
				t.Fatalf("unexpected %s payload: got=%q want=%q", provider, string(got), string(content))
			}

			if err := store.Delete(ctx, uri); err != nil {
				t.Fatalf("delete %s object: %v", provider, err)
			}
			if _, err := store.Get(ctx, uri); err == nil {
				t.Fatalf("expected get %s object after delete to fail", provider)
			}
		})
	}
}

func readFirstNonEmpty(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeEndpoint(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimSuffix(value, "/")
	return value
}
