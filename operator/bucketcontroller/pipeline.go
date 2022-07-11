package bucketcontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	bucketv1 "github.com/vshn/appcat-service-s3/apis/bucket/v1"
	"github.com/vshn/appcat-service-s3/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BucketPipeline provisions ObjectsUsers on cloudscale.ch
type BucketPipeline struct{}

// BucketFinalizer is the name of the finalizer to protect unchecked deletions.
const BucketFinalizer = "s3.appcat.vshn.io/bucket-protection"

// NewBucketPipeline returns a new instance of BucketPipeline.
func NewBucketPipeline() *BucketPipeline {
	return &BucketPipeline{}
}

// Run executes the business logic.
func (p *BucketPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(BucketPipeline{}, BucketFinalizer)),
			pipeline.NewStepFromFunc("set status condition", steps.MarkObjectReadyFn(BucketKey{})),
		).
		WithFinalizer(steps.ErrorHandlerFn(BucketKey{}))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func emitSuccessEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, BucketKey{}).(client.Object)

	recorder.Event(user, corev1.EventTypeNormal, "Created", "Bucket successfully created")
	return nil
}
