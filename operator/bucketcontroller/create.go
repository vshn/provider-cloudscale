package bucketcontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *ProvisioningPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	bucket := fromManaged(mg)
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("create bucket", p.createS3BucketFn(bucket)),
			pipeline.NewStepFromFunc("emit event", p.emitCreationEventFn(bucket)),
		)
	result := pipe.RunWithContext(ctx)

	return managed.ExternalCreation{}, errors.Wrap(result.Err(), "cannot provision bucket")
}

// createS3BucketFn creates a new bucket and sets the name in the status.
// If the bucket already exists, and we have permissions to access it, no error is returned and the name is set in the status.
// If the bucket exists, but we don't own it, an error is returned.
func (p *ProvisioningPipeline) createS3BucketFn(bucket *cloudscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		s3Client := p.minio

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
}

func (p *ProvisioningPipeline) emitCreationEventFn(obj runtime.Object) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		p.recorder.Event(obj, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Created",
			Message: "Bucket successfully created",
		})
		return nil
	}
}
