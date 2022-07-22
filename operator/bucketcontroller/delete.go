package bucketcontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *ProvisioningPipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	bucket := fromManaged(mg)
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("delete bucket", p.deleteS3BucketFn(bucket)),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(bucket)),
		)
	result := pipe.RunWithContext(ctx)
	return errors.Wrap(result.Err(), "cannot deprovision bucket")
}

// deleteS3BucketFn deletes the bucket.
// NOTE: The removal fails if there are still objects in the bucket.
// This func does not recursively delete all objects beforehand.
func (p *ProvisioningPipeline) deleteS3BucketFn(bucket *cloudscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		s3Client := p.minio

		bucketName := bucket.Status.AtProvider.BucketName
		err := s3Client.RemoveBucket(ctx, bucketName)
		return err
	}
}

func (p *ProvisioningPipeline) emitDeletionEventFn(obj runtime.Object) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		p.recorder.Event(obj, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Deleted",
			Message: "Bucket deleted",
		})
		return nil
	}
}
