package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func newTestS3Client(serverURL string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRET_KEY", "TOKEN")),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: serverURL, SigningRegion: region}, nil
			})),
	)
	if err != nil {
		return nil, err
	}
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

func TestS3Storage_Write(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT request, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello world" {
			t.Errorf("expected body 'hello world', got %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newTestS3Client(server.URL)
	if err != nil {
		t.Fatalf("failed to create test s3 client: %v", err)
	}

	storage := &S3Storage{
		client: client,
		bucket: "test-bucket",
	}

	err = storage.Write("test-path", strings.NewReader("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestS3Storage_Read(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	client, err := newTestS3Client(server.URL)
	if err != nil {
		t.Fatalf("failed to create test s3 client: %v", err)
	}

	storage := &S3Storage{
		client: client,
		bucket: "test-bucket",
	}

	rc, err := storage.Read("test-path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if string(body) != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", string(body))
	}
}

func TestS3Storage_List(t *testing.T) {
	response := `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Name>test-bucket</Name>
    <Prefix>test-prefix</Prefix>
    <KeyCount>2</KeyCount>
    <MaxKeys>1000</MaxKeys>
    <IsTruncated>false</IsTruncated>
    <Contents>
        <Key>test-prefix/file1.txt</Key>
        <LastModified>2024-01-01T00:00:00.000Z</LastModified>
        <ETag>"abc"</ETag>
        <Size>123</Size>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
    <Contents>
        <Key>test-prefix/file2.txt</Key>
        <LastModified>2024-01-01T00:00:00.000Z</LastModified>
        <ETag>"def"</ETag>
        <Size>456</Size>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
</ListBucketResult>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	client, err := newTestS3Client(server.URL)
	if err != nil {
		t.Fatalf("failed to create test s3 client: %v", err)
	}

	storage := &S3Storage{
		client: client,
		bucket: "test-bucket",
	}

	paths, err := storage.List("test-prefix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "test-prefix/file1.txt" {
		t.Errorf("expected path 'test-prefix/file1.txt', got '%s'", paths[0])
	}
	if paths[1] != "test-prefix/file2.txt" {
		t.Errorf("expected path 'test-prefix/file2.txt', got '%s'", paths[1])
	}
}
