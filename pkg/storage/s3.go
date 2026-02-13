package storage

import (
	"context"
	"io"
	"net/url"

	borgconfig "github.com/Snider/Borg/pkg/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Storage is a storage backend for S3.
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage creates a new S3 storage backend.
func NewS3Storage(bucket string) (*S3Storage, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	remotes, err := borgconfig.LoadRemotes()
	if err == nil { // Silently ignore errors, fallback to default config
		for _, r := range remotes {
			u, err := url.Parse(r.URL)
			if err != nil {
				continue
			}
			if u.Host == bucket {
				creds := credentials.NewStaticCredentialsProvider(r.AccessKey, r.SecretKey, "")
				cfg.Credentials = creds
				if r.Endpoint != "" {
					cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(
						func(service, region string, options ...interface{}) (aws.Endpoint, error) {
							return aws.Endpoint{URL: r.Endpoint}, nil
						})
				}
				break
			}
		}
	}

	client := s3.NewFromConfig(cfg)
	return &S3Storage{
		client: client,
		bucket: bucket,
	}, nil
}

// Write writes data to the given path.
func (s *S3Storage) Write(path string, data io.Reader) error {
	uploader := s3manager.NewUploader(s.client)
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   data,
	})
	return err
}

// Read reads data from the given path.
func (s *S3Storage) Read(path string) (io.ReadCloser, error) {
	out, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

// List lists the contents of the given path.
func (s *S3Storage) List(path string) ([]string, error) {
	out, err := s.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(path),
	})
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, obj := range out.Contents {
		paths = append(paths, *obj.Key)
	}
	return paths, nil
}
