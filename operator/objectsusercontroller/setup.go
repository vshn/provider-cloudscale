package objectsusercontroller

import (
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:rbac:groups=cloudscale.crossplane.io,resources=objectsusers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cloudscale.crossplane.io,resources=objectsusers/status;objectsusers/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create

// SetupController adds a controller that reconciles cloudscalev1.ObjectsUser managed resources.
func SetupController(mgr ctrl.Manager) error {
	name := strings.ToLower(cloudscalev1.ObjectsUserGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(cloudscalev1.ObjectsUserGroupVersionKind),
		managed.WithExternalConnecter(&objectsUserConnector{
			kube:     mgr.GetClient(),
			recorder: recorder,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(recorder),
		managed.WithPollInterval(1*time.Hour), // object users are rather static
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&cloudscalev1.ObjectsUser{}).
		Complete(r)
}
