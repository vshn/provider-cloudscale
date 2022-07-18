package cloudscale

import (
	"context"
	"fmt"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	bucketv1 "github.com/vshn/provider-cloudscale/apis/bucket/v1"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/apis/conditions"
	"github.com/vshn/provider-cloudscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ProvisioningPipeline provisions ObjectsUsers on cloudscale.ch
type ProvisioningPipeline struct{}

// NewProvisioningPipeline returns a new instance of ProvisioningPipeline.
func NewProvisioningPipeline() *ProvisioningPipeline {
	return &ProvisioningPipeline{}
}

// Run executes the business logic.
func (p *ProvisioningPipeline) Run(ctx context.Context) error {
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("add finalizer", steps.AddFinalizerFn(ObjectsUserKey{}, userFinalizer)),
			pipeline.NewStepFromFunc("create client", CreateCloudscaleClientFn(APIToken)),
			pipeline.IfOrElse(isObjectsUserIDKnown,
				pipeline.NewStepFromFunc("fetch objects user", GetObjectsUser),
				pipeline.NewPipeline().WithNestedSteps("new user",
					pipeline.NewStepFromFunc("create objects user", CreateObjectsUser),
					pipeline.NewStepFromFunc("set user in status", steps.UpdateStatusFn(ObjectsUserKey{})),
					pipeline.NewStepFromFunc("emit event", emitCreationEvent),
				),
			),
			pipeline.NewStepFromFunc("ensure credential secret", ensureCredentialSecret),
			pipeline.NewStepFromFunc("set status condition", steps.MarkObjectReadyFn(ObjectsUserKey{})),
		).
		WithFinalizer(steps.ErrorHandlerFn(ObjectsUserKey{}, conditions.ReasonProvisioningFailed))
	result := pipe.RunWithContext(ctx)
	return result.Err()
}

func isObjectsUserIDKnown(ctx context.Context) bool {
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	return user.Status.UserID != ""
}

func emitCreationEvent(ctx context.Context) error {
	recorder := steps.GetEventRecorderFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(client.Object)

	recorder.Event(user, corev1.EventTypeNormal, "Created", "ObjectsUser successfully created")
	return nil
}

// UserCredentialSecretKey identifies the credential Secret in the context.
type UserCredentialSecretKey struct{}

func ensureCredentialSecret(ctx context.Context) error {
	kube := steps.GetClientFromContext(ctx)
	user := steps.GetFromContextOrPanic(ctx, ObjectsUserKey{}).(*cloudscalev1.ObjectsUser)
	csUser := steps.GetFromContextOrPanic(ctx, CloudscaleUserKey{}).(*cloudscalesdk.ObjectsUser)
	log := controllerruntime.LoggerFrom(ctx)

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: user.Spec.SecretRef, Namespace: user.Namespace}}

	if keyErr := checkUserForKeys(csUser); keyErr != nil {
		return keyErr
	}

	// See https://www.cloudscale.ch/en/api/v1#objects-users

	_, err := controllerruntime.CreateOrUpdate(ctx, kube, secret, func() error {
		secret.Labels = labels.Merge(secret.Labels, getCommonLabels(user.Name))
		if secret.StringData == nil {
			secret.StringData = map[string]string{}
		}
		secret.StringData[bucketv1.AccessKeyIDName] = csUser.Keys[0]["access_key"]
		secret.StringData[bucketv1.SecretAccessKeyName] = csUser.Keys[0]["secret_key"]
		controllerutil.AddFinalizer(secret, userFinalizer)
		return controllerutil.SetOwnerReference(user, secret, kube.Scheme())
	})

	pipeline.StoreInContext(ctx, UserCredentialSecretKey{}, secret)
	return logIfNotError(err, log, 1, "Ensured credential secret", "secretName", user.Spec.SecretRef)
}

func getCommonLabels(instanceName string) labels.Set {
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	return labels.Set{
		"app.kubernetes.io/instance":   instanceName,
		"app.kubernetes.io/managed-by": cloudscalev1.Group,
		"app.kubernetes.io/created-by": fmt.Sprintf("controller-%s", strings.ToLower(cloudscalev1.ObjectsUserKind)),
	}
}
