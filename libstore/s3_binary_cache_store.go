package libstore

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"io"
	"net/url"
)

type S3BinaryCacheStore struct {
	url        *url.URL
	BucketName string
	Client     *s3.S3
}

func NewS3BinaryCacheStore(u *url.URL) (*S3BinaryCacheStore, error) {
	scheme := u.Query().Get("scheme")
	profile := u.Query().Get("profile")
	region := u.Query().Get("region")
	endpoint := u.Query().Get("endpoint")
	bucketName := u.Host

	var disableSSL bool
	switch scheme {
	case "http":
		disableSSL = true
	case "https", "":
		disableSSL = false
	default:
		return &S3BinaryCacheStore{}, fmt.Errorf("Unsupported scheme %s", scheme)
	}

	var sess = session.Must(session.NewSessionWithOptions(session.Options{
		// Specify profile to load for the session's config
		Profile: profile,

		// Provide SDK Config options, such as Region.
		Config: aws.Config{
			Region:           aws.String(region),
			Endpoint:         &endpoint,
			DisableSSL:       aws.Bool(disableSSL),
			S3ForcePathStyle: aws.Bool(true),
		},
	}))

	svc := s3.New(sess)
	return &S3BinaryCacheStore{
		url:        u,
		BucketName: bucketName,
		Client:     svc,
	}, nil
}

func (c *S3BinaryCacheStore) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := c.GetFile(ctx, path)
	aerr, ok := err.(awserr.Error)
	if ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchKey:
			return false, aerr
		default:
			return true, aerr
		}
	} else {
		return true, nil
	}
}

func (c *S3BinaryCacheStore) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.BucketName),
		Key:    aws.String(path),
	}

	obj, err := c.Client.GetObjectWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return obj.Body, nil // for now we return Object data with type blob
}

// URL returns the store URI
func (c S3BinaryCacheStore) URL() string {
	return c.url.String()
}
