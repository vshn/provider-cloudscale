package cloudscale

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/operator/steps"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
)

// ObjectsUserPipeline provisions ObjectsUsers on cloudscale.ch
type ObjectsUserPipeline struct {
}

// NewObjectsUserPipeline returns a new instance of ObjectsUserPipeline.
func NewObjectsUserPipeline() *ObjectsUserPipeline {
	return &ObjectsUserPipeline{}
}

// Run executes the business logic.
func (p *ObjectsUserPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(ObjectsUserKey{}, userFinalizer)),
			pipeline.NewStepFromFunc("validate spec", validateSpec()),
			pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
			pipeline.IfOrElse(isObjectsUserIDKnown(),
				pipeline.NewStepFromFunc("fetch objects user", GetObjectsUserFn()),
				pipeline.NewPipeline().WithNestedSteps("new user",
					pipeline.NewStepFromFunc("create objects user", CreateObjectsUserFn()),
					pipeline.NewStepFromFunc("set user in status", steps.UpdateStatusFn(ObjectsUserKey{})),
				),
			),
			pipeline.NewStepFromFunc("ensure credential secret", EnsureCredentialSecretFn()),
			pipeline.NewStepFromFunc("set status condition", markUserReadyFn()),
		).
		)
	result := pipe.RunWithContext(ctx)
	if result.IsFailed() {
		// TODO: Add failed condition
		// TODO: Emit event
		return result.Err()
	}
	return nil
}

func validateSpec() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
		if user.Spec.SecretRef == "" {
			return fmt.Errorf("spec.secretRef cannot be empty")
		}
		return nil
	}
}

func isObjectsUserIDKnown() func(ctx context.Context) bool {
	return func(ctx context.Context) bool {
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
		return user.Status.UserID != ""
	}
}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": cloudscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(cloudscalev1.ObjectsUserKind)),
	}
}

func markUserReadyFn() func(ctx context.Context) error {
	return func(ctx context.Context) error {
		kube := steps.GetClientFromContext(ctx)
		user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)

		meta.SetStatusCondition(&user.Status.Conditions, conditions.Ready())
		meta.RemoveStatusCondition(&user.Status.Conditions, conditions.TypeFailed)
		return kube.Status().Update(ctx, user)
	}

}
