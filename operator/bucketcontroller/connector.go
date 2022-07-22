package bucketcontroller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bucketConnector struct {
	kube     client.Client
	recorder event.Recorder

	bucket            *cloudscalev1.Bucket
	credentialsSecret *corev1.Secret
	minio             *minio.Client
}

// Connect implements managed.ExternalConnecter.
func (c *bucketConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	bucket := fromManaged(mg)
	c.bucket = bucket
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("fetch secret", c.fetchCredentialsSecret),
			pipeline.NewStepFromFunc("validate secret", c.validateSecret),
			pipeline.NewStepFromFunc("create S3 client", c.createS3Client),
		)
	result := pipe.RunWithContext(ctx)

	if result.IsFailed() {
		return nil, result.Err()
	}

	// We don't need anything from a ProviderConfig.
	// The S3 credentials are loaded as part of the CRUD methods.
	return NewProvisioningPipeline(c.kube, c.recorder, c.minio), nil
}

func (c *bucketConnector) fetchCredentialsSecret(ctx context.Context) error {
	bucket := c.bucket
	log := controllerruntime.LoggerFrom(ctx)

	c.credentialsSecret = &corev1.Secret{}
	secretRef := bucket.Spec.ForProvider.CredentialsSecretRef
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, c.credentialsSecret)
	if err != nil {
		return err
	}
	log.V(1).Info("Fetched credentials secret", "secret name", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil
}

func (c *bucketConnector) validateSecret(_ context.Context) error {
	secret := c.credentialsSecret

	if secret.Data == nil {
		return fmt.Errorf("secret %q does not have any data", fmt.Sprintf("%s/%s", secret.Namespace, secret.Name))
	}

	requiredKeys := []string{cloudscalev1.AccessKeyIDName, cloudscalev1.SecretAccessKeyName}
	for _, key := range requiredKeys {
		if val, exists := secret.Data[key]; !exists || string(val) == "" {
			return fmt.Errorf("secret %q is missing one of the following keys or content: %s", fmt.Sprintf("%s/%s", secret.Namespace, secret.Name), requiredKeys)
		}
	}
	return nil
}

// createS3Client creates a new client using the S3 credentials from the Secret.
func (c *bucketConnector) createS3Client(_ context.Context) error {
	bucket := c.bucket
	secret := c.credentialsSecret

	parsed, err := url.Parse(bucket.Spec.ForProvider.EndpointURL)
	if err != nil {
		return err
	}

	// we assume here that the secret has the expected keys and data.
	accessKey := string(secret.Data[cloudscalev1.AccessKeyIDName])
	secretKey := string(secret.Data[cloudscalev1.SecretAccessKeyName])

	host := parsed.Host
	if parsed.Host == "" {
		host = parsed.Path // if no scheme is given, it's parsed as a path -.-
	}
	s3Client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: isTLSEnabled(parsed),
	})
	c.minio = s3Client
	return err
}

// isTLSEnabled returns false if the scheme is explicitly set to `http` or `HTTP`
func isTLSEnabled(u *url.URL) bool {
	return !strings.EqualFold(u.Scheme, "http")
}
