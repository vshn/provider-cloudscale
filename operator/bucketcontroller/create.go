package bucketcontroller

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/pipelineutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create implements managed.ExternalClient.
func (p *ProvisioningPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	bucket := fromManaged(mg)
	pctx := &pipelineContext{Context: ctx, Bucket: bucket}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("create bucket", p.createS3Bucket),
			pipe.NewStep("fetch objects user connection secret", p.fetchSecret),
			pipe.NewStep("create bucket connection secret", p.createBucketConnectionSecret),
			pipe.NewStep("emit event", p.emitCreationEvent),
		)
	err := pipe.RunWithContext(pctx)

	return managed.ExternalCreation{}, errors.Wrap(err, "cannot provision bucket")
}

// createS3Bucket creates a new bucket and sets the name in the status.
// If the bucket already exists, and we have permissions to access it, no error is returned and the name is set in the status.
// If the bucket exists, but we don't own it, an error is returned.
func (p *ProvisioningPipeline) createS3Bucket(ctx *pipelineContext) error {
	s3Client := p.minio
	bucket := ctx.Bucket

	bucketName := bucket.GetBucketName()
	err := s3Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: bucket.Spec.ForProvider.Region})

	if err != nil {
		// Check to see if we already own this bucket (which happens if we run this twice)
		exists, errBucketExists := s3Client.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			return nil
		} else {
			// someone else might have created the bucket
			return err
		}
	}
	return nil
}

// createBucketConnectionSecret creates a new secret for bucket with objects user credentials, EndpointName, RegionName and BucketName
func (p *ProvisioningPipeline) createBucketConnectionSecret(ctx *pipelineContext) error {
	bucket := ctx.Bucket
	log := controllerruntime.LoggerFrom(ctx)

	secretRef := bucket.Spec.WriteConnectionSecretToReference

	credentialSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretRef.Name, Namespace: secretRef.Namespace}}
	_, err := controllerruntime.CreateOrUpdate(ctx, p.kube, credentialSecret, func() error {
		credentialSecret.Data = map[string][]byte{}
		credentialSecret.Data[cloudscalev1.AccessKeyIDName] = ctx.ObjectsUserCredentialSecret.Data[cloudscalev1.AccessKeyIDName]
		credentialSecret.Data[cloudscalev1.SecretAccessKeyName] = ctx.ObjectsUserCredentialSecret.Data[cloudscalev1.SecretAccessKeyName]
		credentialSecret.Data[EndpointName] = []byte(bucket.Spec.ForProvider.EndpointURL)
		credentialSecret.Data[RegionName] = []byte(bucket.Spec.ForProvider.Region)
		credentialSecret.Data[BucketName] = []byte(bucket.GetBucketName())
		return controllerutil.SetOwnerReference(bucket, credentialSecret, p.kube.Scheme())
	})
	if err != nil {
		return err
	}
	log.V(1).Info("Enriched credential secret", "secretName", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil
}

func (p *ProvisioningPipeline) emitCreationEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.Bucket, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Created",
		Message: "Bucket successfully created",
	})
	return nil
}
