package blob

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
)

func New(ctx context.Context, bucket string) (*S3, error) {
	sdkConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load s3 config: %w", err)
	}

	client := &S3{
		bucket: bucket,
		s3:     s3.NewFromConfig(sdkConfig),
	}

	if err := client.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return client, nil
}

type S3 struct {
	bucket string
	s3     *s3.Client
}

func (c *S3) Get(ctx context.Context, path string) (*s3.GetObjectOutput, error) {
	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return resp, nil
}

type PutRequest struct {
	Key           string
	Body          io.Reader
	ContentLength int64
	ContentType   string
	Metadata      map[string]string
}

func (c *S3) Put(ctx context.Context, req PutRequest) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(req.Key),
		ContentLength: req.ContentLength,
		ContentType:   aws.String(req.ContentType),
		Body:          req.Body,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	return nil
}

func (c *S3) ensureBucket(ctx context.Context) error {
	bucket := aws.String(c.bucket)

	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: bucket,
	})
	switch {
	case err == nil:
		// Bucket already exists
		return nil
	case err != nil && strings.Contains(err.Error(), "NotFound"):
		log.Printf("creating bucket %q\n", c.bucket)
		{
			_, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: bucket,
				CreateBucketConfiguration: &types.CreateBucketConfiguration{
					LocationConstraint: types.BucketLocationConstraint(
						os.Getenv("AWS_REGION"),
					),
				},
			})
			if err != nil {
				return fmt.Errorf("create bucket: %w", err)
			}
			return nil
		}
	default:
		return fmt.Errorf("head bucket: %w", err)
	}
}
