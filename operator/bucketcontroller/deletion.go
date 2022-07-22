package bucketcontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	bucketv1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/apis/conditions"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeletionPipeline deletes buckets.
type DeletionPipeline struct{}

// NewDeletionPipeline returns a new DeletionPipeline.
func NewDeletionPipeline() *DeletionPipeline {
	return &DeletionPipeline{}
}

// Run executes the business logic.
func (p *DeletionPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.If(bucketExisting, pipeline.NewPipeline().WithNestedSteps("deprovision bucket",
				pipeline.NewStepFromFunc("fetch credentials secret", fetchCredentialsSecret),
				pipeline.NewStepFromFunc("validate secret", validateSecret),
				pipeline.NewStepFromFunc("create S3 client", CreateS3Client),
				pipeline.NewStepFromFunc("delete bucket", DeleteS3Bucket),
				pipeline.NewStepFromFunc("emit event", emitDeletionEvent),
			)),
			pipeline.NewStepFromFunc("remove finalizer", steps.RemoveFinalizerFn(BucketKey{}, BucketFinalizer)),
		).
		WithFinalizer(steps.ErrorHandlerFn(BucketKey{}, conditions.ReasonDeletionFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func bucketExisting(ctx context.Context) bool {
	bucket := steps.GetFromContextOrPanic(ctx, BucketKey{}).(*bucketv1.Bucket)

	return bucket.Status.AtProvider.BucketName != ""
}

func emitDeletionEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	obj := steps.GetFromContextOrPanic(ctx, BucketKey{}).(client.Object)

	recorder.Event(obj, corev1.EventTypeNormal, "Deleted", "Bucket deleted")
	return nil
}
