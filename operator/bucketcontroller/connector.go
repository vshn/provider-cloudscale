package bucketcontroller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bucketConnector struct {
	kube     client.Client
	recorder event.Recorder
}

type connectContext struct {
	context.Context
	bucket            *cloudscalev1.Bucket
	minio             *minio.Client
	credentialsSecret *corev1.Secret
}

// Connect implements managed.ExternalConnecter.
func (c *bucketConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	bucket := fromManaged(mg)

	if isBucketAlreadyDeleted(bucket) {
		// See https://github.com/vshn/provider-cloudscale/issues/24 for full explanation as to why.
		// We do this to prevent after-deletion reconciliations since there is a chance that we might not have the access credentials anymore.
		log.V(1).Info("Bucket already deleted, skipping observation")
		return &NoopClient{}, nil
	}

	bucket.Status.Endpoint = fmt.Sprintf("objects.%s.cloudscale.ch", bucket.Spec.ForProvider.Region)
	bucket.Status.EndpointURL = fmt.Sprintf("https://%s", bucket.Status.Endpoint)
	pctx := &connectContext{Context: ctx, bucket: bucket}
	pipe := pipeline.NewPipeline[*connectContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("fetch secret", c.fetchCredentialsSecret),
			pipe.NewStep("validate secret", c.validateSecret),
			pipe.NewStep("create S3 client", c.createS3Client),
		)
	result := pipe.RunWithContext(pctx)

	if result != nil {
		return nil, result
	}

	// We don't need anything from a ProviderConfig.
	// The S3 credentials are loaded as part of the CRUD methods.
	return NewProvisioningPipeline(c.kube, c.recorder, pctx.minio), nil
}

func (c *bucketConnector) fetchCredentialsSecret(ctx *connectContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	bucket := ctx.bucket

	secret := &corev1.Secret{}
	secretRef := bucket.Spec.ForProvider.CredentialsSecretRef
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, secret)
	if err != nil {
		return err
	}
	ctx.credentialsSecret = secret
	log.V(1).Info("Fetched credentials secret", "secret name", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil
}

func (c *bucketConnector) validateSecret(ctx *connectContext) error {
	secret := ctx.credentialsSecret

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
func (c *bucketConnector) createS3Client(ctx *connectContext) error {
	secret := ctx.credentialsSecret
	bucket := ctx.bucket

	parsed, err := url.Parse(bucket.Status.EndpointURL)
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
	ctx.minio = s3Client
	return err
}

// isBucketAlreadyDeleted returns true if the status conditions are in a state where one can assume that the deletion of a bucket was successful in a previous reconciliation.
// This is useful to prevent further reconciliation with possibly lost S3 credentials.
func isBucketAlreadyDeleted(bucket *cloudscalev1.Bucket) bool {
	readyCond := findCondition(bucket.Status.Conditions, xpv1.TypeReady)
	syncCond := findCondition(bucket.Status.Conditions, xpv1.TypeSynced)

	if readyCond != nil && syncCond != nil {
		// These 4 criteria must be in exactly this combination
		if readyCond.Status == corev1.ConditionFalse &&
			readyCond.Reason == xpv1.ReasonDeleting &&
			syncCond.Status == corev1.ConditionTrue &&
			syncCond.Reason == xpv1.ReasonReconcileSuccess {
			return true
		}
	}
	return false
}

func findCondition(conds []xpv1.Condition, condType xpv1.ConditionType) *xpv1.Condition {
	for _, cond := range conds {
		if cond.Type == condType {
			return &cond
		}
	}
	return nil
}

// isTLSEnabled returns false if the scheme is explicitly set to `http` or `HTTP`
func isTLSEnabled(u *url.URL) bool {
	return !strings.EqualFold(u.Scheme, "http")
}
