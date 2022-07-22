package bucketcontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/go-logr/logr"
	bucketv1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"github.com/vshn/provider-cloudscale/operator/steps"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// +kubebuilder:rbac:groups=s3.appcat.vshn.io,resources=buckets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=s3.appcat.vshn.io,resources=buckets/status;buckets/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create

// BucketReconciler reconciles Bucket resources.
type BucketReconciler struct {
	client   client.Client
	recorder record.EventRecorder
}

// BucketKey identifies a bucketv1.Bucket in the context.
type BucketKey struct{}

// Reconcile implements reconcile.Reconciler.
func (r *BucketReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Reconciling")
	obj := &bucketv1.Bucket{}
	err := r.client.Get(ctx, request.NamespacedName, obj)
	if err != nil && apierrors.IsNotFound(err) {
		// doesn't exist anymore, nothing to do
		return reconcile.Result{}, nil
	}
	if err != nil {
		// some other error
		return reconcile.Result{}, err
	}
	pipeline.StoreInContext(ctx, BucketKey{}, obj)
	steps.SetClientInContext(ctx, r.client)
	steps.SetEventRecorderInContext(ctx, r.recorder)
	if !obj.DeletionTimestamp.IsZero() {
		return r.Delete(ctx)
	}
	return r.Provision(ctx)
}

// Provision reconciles the given object.
func (r *BucketReconciler) Provision(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Provisioning resource")
	p := NewProvisioningPipeline()
	err := p.Run(ctx)
	return reconcile.Result{}, err
}

// Delete prepares the given object for deletion.
func (r *BucketReconciler) Delete(ctx context.Context) (reconcile.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Deleting resource")
	p := NewDeletionPipeline()
	err := p.Run(ctx)
	return reconcile.Result{Requeue: true}, err
}

func logIfNotError(err error, log logr.Logger, level int, msg string, keysAndValues ...any) error {
	if err != nil {
		return err
	}
	log.V(level).Info(msg, keysAndValues...)
	return nil
}
