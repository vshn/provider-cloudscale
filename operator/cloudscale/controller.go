package cloudscale

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/go-logr/logr"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"github.com/vshn/appcat-service-s3/operator/steps"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var userFinalizer = "s3.appcat.vshn.io/user-protection"

// +kubebuilder:rbac:groups=cloudscale.s3.appcat.vshn.io,resources=objectsusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudscale.s3.appcat.vshn.io,resources=objectsusers/status;objectsusers/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create

// ObjectsUserReconciler reconciles cloudscalev1.ObjectsUser.
type ObjectsUserReconciler struct {
	client   client.Client
	recorder record.EventRecorder
}

// ObjectsUserKey identifies the ObjectsUser in the context.
type ObjectsUserKey struct{}

// Reconcile implements reconcile.Reconciler.
func (r *ObjectsUserReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Reconciling")
	obj := &cloudscalev1.ObjectsUser{}
	err := r.client.Get(ctx, request.NamespacedName, obj)
	if err != nil && apierrors.IsNotFound(err) {
		// doesn't exist anymore, nothing to do
		return reconcile.Result{}, nil
	}
	if err != nil {
		// some other error
		return reconcile.Result{}, err
	}
	pipeline.StoreInContext(ctx, ObjectsUserKey{}, obj)
	steps.SetClientInContext(ctx, r.client)
	steps.SetEventRecorderInContext(ctx, r.recorder)
	if !obj.DeletionTimestamp.IsZero() {
		return r.Delete(ctx)
	}
	return r.Provision(ctx)
}

// Provision reconciles the given object.
func (r *ObjectsUserReconciler) Provision(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Provisioning resource")
	p := NewObjectsUserPipeline()
	err := p.Run(ctx)
	return reconcile.Result{}, err
}

// Delete prepares the given object for deletion.
func (r *ObjectsUserReconciler) Delete(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting resource")
	return reconcile.Result{Requeue: true}, nil
}

func logIfNotError(err error, log logr.Logger, level int, msg string, keysAndValues ...any) error {
	if err != nil {
		return err
	}
	log.V(level).Info(msg, keysAndValues...)
	return nil
}
