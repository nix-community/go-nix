package libstore

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"io"
	"log"
	"net/url"
)

type S3Store struct {
	Profile    string
	Region     string
	Scheme     string
	BucketName string
	Client     *s3.S3
}

func (s *S3Store) FileExists(ctx context.Context, path string) (bool, error) {
	_, err := s.GetFile(ctx, path)
	if err != nil {
		return false, nil
	}
	return true, err
}

func (s *S3Store) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.BucketName),
		Key: aws.String(path),
	}

	obj, err := s.Client.GetObjectWithContext(ctx, input)
	if err != nil {
    if aerr, ok := err.(awserr.Error); ok {
        switch aerr.Code() {
        case s3.ErrCodeNoSuchKey:
            fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
        default:
            fmt.Println(aerr.Error())
        }
    } else {
        // Print the error, cast err to awserr.Error to get the Code and
        // Message from an error.
        fmt.Println(err.Error())
    }
    return nil, err
	}

	return obj.Body, nil // for now we return Object data with type blob
}

func NewStoreReader(uri string) (*S3Store, error) {
	// example: s3://example-nix-cache?profile=cache-upload&scheme=https&endpoint=minio.example.com&region=eu-west-2
	u, err := url.Parse(uri)
	if err != nil {
		log.Fatal(err)
	}

	if u.Scheme != "s3" {
		fmt.Errorf("Invalid S3 URL")
		return nil, err
	}

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
		return &S3Store{}, fmt.Errorf("Unsupported scheme %s", scheme)
	}

	var sess = session.Must(session.NewSessionWithOptions(session.Options{
		// Specify profile to load for the session's config
		Profile: profile,

		// Provide SDK Config options, such as Region.
		Config: aws.Config{
			Region:     aws.String(region),
			Endpoint:   &endpoint,
			DisableSSL: aws.Bool(disableSSL),
			S3ForcePathStyle: aws.Bool(true),

		},
	}))

	svc := s3.New(sess)
	return &S3Store{
		Profile:    profile,
		Region:     region,
		Scheme:     scheme,
		BucketName: bucketName,
		Client:     svc,
	}, nil
}
