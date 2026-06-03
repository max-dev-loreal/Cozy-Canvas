package storage

import (
	"context"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOClient struct {
	client     *minio.Client
	bucketName string
}

func NewMinIOClient(endpoint, accessKey, secretKey, bucketName string, secure bool) (*MinIOClient, error) {
	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &MinIOClient{
		client:     minioClient,
		bucketName: bucketName,
	}, nil
}

func (m *MinIOClient) GeneratePresignedPutURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	u, err := m.client.PresignedPutObject(ctx, m.bucketName, objectName, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (m *MinIOClient) GeneratePresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	u, err := m.client.PresignedGetObject(ctx, m.bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
