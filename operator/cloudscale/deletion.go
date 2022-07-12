package cloudscale

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/apis/conditions"
	"github.com/vshn/appcat-service-s3/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeletionPipeline deletes ObjectsUsers on cloudscale.ch
type DeletionPipeline struct{}

// NewDeletionPipeline returns a new instance of DeletionPipeline.
func NewDeletionPipeline() *DeletionPipeline {
	return &DeletionPipeline{}
}

// Run executes the business logic.
func (p *DeletionPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.If(isObjectsUserIDKnown, pipeline.NewPipeline().WithNestedSteps("deprovision objects user",
				pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
				pipeline.NewStepFromFunc("delete objects user", DeleteObjectsUser),
				pipeline.NewStepFromFunc("fetch credentials secret", fetchCredentialsSecret),
				// Note: We do not need to check if there are Bucket resources still requiring the Secret.
				// Cloudscale's API returns an error if there are still buckets existing for that user, which ultimately also ends up as a Failed condition in the ObjectsUser resource.
				pipeline.NewStepFromFunc("delete finalizer from secret", steps.RemoveFinalizerFn(UserCredentialSecretKey{}, userFinalizer)),
				pipeline.NewStepFromFunc("emit event", emitDeletionEvent),
			)),
			pipeline.NewStepFromFunc("remove finalizer", steps.RemoveFinalizerFn(ObjectsUserKey{}, userFinalizer)),
		).
		WithFinalizer(steps.ErrorHandlerFn(ObjectsUserKey{}, conditions.ReasonDeletionFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func fetchCredentialsSecret(ctx context.Context) error {
	kube := steps.GetClientFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	secret := &corev1.Secret{}
	err := kube.Get(ctx, types.NamespacedName{Name: user.Spec.SecretRef, Namespace: user.Namespace}, secret)
	pipeline.StoreInContext(ctx, UserCredentialSecretKey{}, secret)
	return logIfNotError(err, log, 1, "Fetched credentials secret", "secretName", user.Spec.SecretRef)
}

func emitDeletionEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(client.Object)

	recorder.Event(user, corev1.EventTypeNormal, "Deleted", "ObjectsUser deleted")
	return nil
}
