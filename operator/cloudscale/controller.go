package cloudscale

import (
	"context"
	cloudscalev1 "github.com/vshn/appcat-service-s3/apis/cloudscale/v1"
	"time"

	pipeline "github.com/ccremer/go-command-pipeline"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var userFinalizer = "s3.appcat.vshn.io/user-protection"

// +kubebuilder:rbac:groups=cloudscale.s3.appcat.vshn.io,resources=objectsusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudscale.s3.appcat.vshn.io,resources=objectsusers/status;objectsusers/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update

// PostgresStandaloneReconciler reconciles cloudscalev1.ObjectsUser.
type PostgresStandaloneReconciler struct {
	client client.Client
}

// Reconcile implements reconcile.Reconciler.
func (r *PostgresStandaloneReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	ctx = pipeline.MutableContext(ctx)
	obj := &cloudscalev1.ObjectsUser{}
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Reconciling")
	err := r.client.Get(ctx, request.NamespacedName, obj)
	if err != nil && apierrors.IsNotFound(err) {
		// doesn't exist anymore, nothing to do
		return reconcile.Result{}, nil
	}
	if err != nil {
		// some other error
		return reconcile.Result{}, err
	}
	if !obj.DeletionTimestamp.IsZero() {
		return r.Delete(ctx)
	}
	return r.Provision(ctx, obj)
}

// Provision reconciles the given object.
func (r *PostgresStandaloneReconciler) Provision(ctx context.Context, instance *cloudscalev1.ObjectsUser) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Provisioning instance")
	return reconcile.Result{}, nil
}

// Delete prepares the given object for deletion.
func (r *PostgresStandaloneReconciler) Delete(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting instance")
	return reconcile.Result{RequeueAfter: 1 * time.Second}, nil
}
