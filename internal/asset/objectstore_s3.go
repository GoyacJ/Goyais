package asset

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"goyais/internal/command"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3CompatibleStore struct {
	provider string
	bucket   string
	region   string
	client   *minio.Client

	initErr error

	bucketOnce sync.Once
	bucketErr  error
}

func NewS3CompatibleStore(options ObjectStoreOptions) ObjectStore {
	provider := strings.ToLower(strings.TrimSpace(options.Provider))
	if provider == "" {
		provider = "minio"
	}

	endpoint := strings.TrimSpace(options.Endpoint)
	switch {
	case endpoint != "":
	case provider == "s3":
		endpoint = "s3.amazonaws.com"
	default:
		endpoint = "127.0.0.1:9000"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
		Secure: options.UseSSL,
		Region: strings.TrimSpace(options.Region),
	})
	if err != nil {
		return &S3CompatibleStore{
			provider: provider,
			bucket:   strings.TrimSpace(options.Bucket),
			region:   strings.TrimSpace(options.Region),
			initErr:  fmt.Errorf("init object store client: %w", err),
		}
	}

	return &S3CompatibleStore{
		provider: provider,
		bucket:   strings.TrimSpace(options.Bucket),
		region:   strings.TrimSpace(options.Region),
		client:   client,
	}
}

func (s *S3CompatibleStore) Put(ctx context.Context, req command.RequestContext, hash string, data []byte, now time.Time) (string, error) {
	if strings.TrimSpace(hash) == "" {
		return "", fmt.Errorf("%w: empty hash", ErrObjectStoreFail)
	}
	if err := s.ensureBucket(ctx); err != nil {
		return "", err
	}

	key := path.Clean(path.Join(
		safeObjectPath(req.TenantID),
		safeObjectPath(req.WorkspaceID),
		now.UTC().Format("2006/01/02"),
		strings.ToLower(strings.TrimSpace(hash)),
	))
	if strings.HasPrefix(key, "../") || strings.Contains(key, "/../") {
		return "", fmt.Errorf("%w: invalid object key", ErrObjectStoreFail)
	}

	if _, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		return "", fmt.Errorf("%w: put object: %v", ErrObjectStoreFail, err)
	}

	return fmt.Sprintf("%s://%s/%s", s.provider, s.bucket, key), nil
}

func (s *S3CompatibleStore) Get(ctx context.Context, uri string) ([]byte, error) {
	if err := s.ensureBucket(ctx); err != nil {
		return nil, err
	}
	bucket, key, err := parseObjectURI(uri)
	if err != nil {
		return nil, err
	}
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: get object: %v", ErrObjectStoreFail, err)
	}
	defer obj.Close()

	raw, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("%w: read object: %v", ErrObjectStoreFail, err)
	}
	return raw, nil
}

func (s *S3CompatibleStore) Delete(ctx context.Context, uri string) error {
	if err := s.ensureBucket(ctx); err != nil {
		return err
	}
	bucket, key, err := parseObjectURI(uri)
	if err != nil {
		return err
	}
	if err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("%w: delete object: %v", ErrObjectStoreFail, err)
	}
	return nil
}

func (s *S3CompatibleStore) Ping(ctx context.Context) error {
	return s.ensureBucket(ctx)
}

func (s *S3CompatibleStore) Provider() string {
	return s.provider
}

func (s *S3CompatibleStore) ensureBucket(ctx context.Context) error {
	if s.initErr != nil {
		return fmt.Errorf("%w: %v", ErrObjectStoreFail, s.initErr)
	}
	if s.client == nil {
		return fmt.Errorf("%w: object store client not initialized", ErrObjectStoreFail)
	}
	if strings.TrimSpace(s.bucket) == "" {
		return fmt.Errorf("%w: empty bucket", ErrObjectStoreFail)
	}

	s.bucketOnce.Do(func() {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err != nil {
			s.bucketErr = err
			return
		}
		if exists {
			return
		}
		makeErr := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region})
		if makeErr == nil {
			return
		}
		resp := minio.ToErrorResponse(makeErr)
		if resp.Code == "BucketAlreadyOwnedByYou" || resp.Code == "BucketAlreadyExists" {
			return
		}
		s.bucketErr = makeErr
	})

	if s.bucketErr != nil {
		return fmt.Errorf("%w: bucket check: %v", ErrObjectStoreFail, s.bucketErr)
	}
	return nil
}

func parseObjectURI(raw string) (bucket string, key string, err error) {
	u, parseErr := url.Parse(strings.TrimSpace(raw))
	if parseErr != nil {
		return "", "", fmt.Errorf("%w: invalid object uri: %v", ErrObjectStoreFail, parseErr)
	}
	if strings.TrimSpace(u.Scheme) == "" || strings.TrimSpace(u.Host) == "" {
		return "", "", fmt.Errorf("%w: invalid object uri", ErrObjectStoreFail)
	}

	key = strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(u.Path)), "/")
	if key == "" || key == "." {
		return "", "", fmt.Errorf("%w: invalid object uri path", ErrObjectStoreFail)
	}
	return strings.TrimSpace(u.Host), key, nil
}

func safeObjectPath(value string) string {
	clean := strings.TrimSpace(value)
	clean = strings.ReplaceAll(clean, "..", "")
	clean = strings.ReplaceAll(clean, "/", "_")
	if clean == "" {
		return "unknown"
	}
	return clean
}
