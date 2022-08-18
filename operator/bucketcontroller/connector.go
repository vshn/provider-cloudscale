package bucketcontroller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type bucketConnector struct {
	ProvisioningPipeline
}

type connectContext struct {
	minio *minio.Client
	pipelineContext
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

	pctx := &connectContext{
		pipelineContext: pipelineContext{
			Context: ctx,
			Bucket:  bucket,
		},
	}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(&pctx.pipelineContext)).
		WithSteps(
			pipe.NewStep("fetch secret", c.fetchSecret),
			pipe.NewStep("validate secret", c.validateSecretFn(pctx)),
			pipe.NewStep("create S3 client", c.createS3ClientFn(pctx)),
		)
	result := pipe.RunWithContext(&pctx.pipelineContext)

	if result != nil {
		return nil, result
	}

	// We don't need anything from a ProviderConfig.
	// The S3 credentials are loaded as part of the CRUD methods.
	return NewProvisioningPipeline(c.kube, c.recorder, pctx.minio), nil
}

func (c *bucketConnector) validateSecretFn(ctx *connectContext) func(pipelineContext *pipelineContext) error {
	return func(_ *pipelineContext) error {
		secret := ctx.ObjectsUserCredentialSecret

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
}

// createS3Client creates a new client using the S3 credentials from the Secret.
func (c *bucketConnector) createS3ClientFn(ctx *connectContext) func(pipelineContext *pipelineContext) error {
	return func(_ *pipelineContext) error {
		secret := ctx.ObjectsUserCredentialSecret
		bucket := ctx.Bucket

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
		ctx.minio = s3Client
		return err
	}
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
