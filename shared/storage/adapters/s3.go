package adapters

import (
	"context"
	"errors"
	"net/http"
	"time"

	"auth_service/shared/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const s3CallTimeout = 15 * time.Second

type S3Options struct {
	Bucket          string
	Region          string
	AccessKeyId     string
	SecretAccessKey string
}

type S3Adapter struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
}

var _ storage.IStorage = &S3Adapter{}

func NewS3Adapter(options S3Options) *S3Adapter {
	client := s3.New(s3.Options{
		Region:      options.Region,
		Credentials: credentials.NewStaticCredentialsProvider(options.AccessKeyId, options.SecretAccessKey, ""),
	})

	return &S3Adapter{
		bucket:    options.Bucket,
		client:    client,
		presigner: s3.NewPresignClient(client),
	}
}

// PresignUpload signs locally - no request leaves the process. S3 enforces every
// condition when the browser posts, against the permissions of the signing key.
func (this *S3Adapter) PresignUpload(key string, contentType string, policy storage.Policy) (*storage.UploadGrant, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s3CallTimeout)
	defer cancel()

	expiresAt := time.Now().Add(policy.GrantTtl).UTC()

	request, err := this.presigner.PresignPostObject(
		ctx,
		&s3.PutObjectInput{Bucket: aws.String(this.bucket), Key: aws.String(key)},
		func(options *s3.PresignPostOptions) {
			options.Expires = policy.GrantTtl
			options.Conditions = []any{
				map[string]string{"key": key},
				map[string]string{"Content-Type": contentType},
				map[string]string{"Cache-Control": policy.CacheControl},
				[]any{"content-length-range", 1, policy.MaxBytes},
			}
		},
	)

	if err != nil {
		return nil, err
	}

	fields := request.Values
	fields["Content-Type"] = contentType
	fields["Cache-Control"] = policy.CacheControl

	return &storage.UploadGrant{
		Key:       key,
		Url:       request.URL,
		Fields:    fields,
		ExpiresAt: expiresAt,
		MaxBytes:  policy.MaxBytes,
	}, nil
}

// A missing object answers 404 only because the key holds s3:ListBucket on the
// prefix; without it S3 answers 403 and this reports an error instead.
func (this *S3Adapter) Head(key string) (*storage.Object, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s3CallTimeout)
	defer cancel()

	output, err := this.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(this.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var response *awshttp.ResponseError

		if errors.As(err, &response) && response.HTTPStatusCode() == http.StatusNotFound {
			return nil, nil
		}

		return nil, err
	}

	return &storage.Object{
		Size:        aws.ToInt64(output.ContentLength),
		ContentType: aws.ToString(output.ContentType),
	}, nil
}

// The copy runs inside S3 and keeps Content-Type and Cache-Control: the
// default metadata directive is COPY.
func (this *S3Adapter) Copy(source string, destination string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3CallTimeout)
	defer cancel()

	_, err := this.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(this.bucket),
		CopySource: aws.String(this.bucket + "/" + source),
		Key:        aws.String(destination),
	})

	return err
}

func (this *S3Adapter) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s3CallTimeout)
	defer cancel()

	_, err := this.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(this.bucket),
		Key:    aws.String(key),
	})

	return err
}
