package bucketcontroller

import (
	"context"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	bucketv1 "github.com/vshn/appcat-service-s3/apis/bucket/v1"
	"github.com/vshn/appcat-service-s3/operator/steps"
	corev1 "k8s.io/api/core/v1"
)

// S3ClientKey identifies the S3 client in the context.
type S3ClientKey struct{}

// CreateS3Client creates a new client using the S3 credentials from the Secret.
func CreateS3Client(ctx context.Context) error {
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*bucketv1.Bucket)
	secret := steps.GetFromContextOrPanic(ctx, CredentialsSecretKey{}).(*corev1.Secret)

	parsed, err := url.Parse(bucket.Spec.EndpointURL)
	if err != nil {
		return err
	}

	// we assume here that the secret has the expected keys and data.
	accessKey := string(secret.Data[bucketv1.AccessKeyIDName])
	secretKey := string(secret.Data[bucketv1.SecretAccessKeyName])

	host := parsed.Host
	if parsed.Host == "" {
		host = parsed.Path // if no scheme is given, it's parsed as a path -.-
	}
	s3Client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: isTLSEnabled(parsed),
	})
	pipeline.StoreInContext(ctx, S3ClientKey{}, s3Client)
	return err
}

// isTLSEnabled returns false if the scheme is explicitly set to `http` or `HTTP`
func isTLSEnabled(u *url.URL) bool {
	return !strings.EqualFold(u.Scheme, "http")
}

// CreateS3Bucket creates a new bucket and sets the name in the status.
// If the bucket already exists, and we have permissions to access it, no error is returned and the name is set in the status.
// If the bucket exists, but we don't own it, an error is returned.
func CreateS3Bucket(ctx context.Context) error {
	s3Client := steps.GetFromContextOrPanic(ctx, S3ClientKey{}).(*minio.Client)
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*bucketv1.Bucket)

	bucketName := bucket.GetBucketName()
	err := s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: bucket.Spec.Region})

	if err != nil {
		// Check to see if we already own this bucket (which happens if we run this twice)
		exists, errBucketExists := s3Client.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			bucket.Status.BucketName = bucketName
			return nil
		} else {
			// someone else might have created the bucket
			return err
		}
	}
	bucket.Status.BucketName = bucketName
	return nil
}
