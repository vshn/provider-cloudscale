package configcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	providerv1 "github.com/vshn/provider-cloudscale/apis/provider/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// SetupController adds a controller that reconciles ProviderConfigs by accounting for their current usage.
func SetupController(mgr ctrl.Manager) error {
	name := providerconfig.ControllerName(providerv1.ProviderConfigGroupKind)

	of := resource.ProviderConfigKinds{
		Config:    providerv1.ProviderConfigGroupVersionKind,
		Usage:     providerv1.ProviderConfigUsageGroupVersionKind,
		UsageList: providerv1.ProviderConfigUsageListGroupVersionKind,
	}

	r := providerconfig.NewReconciler(mgr, of,
		providerconfig.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&providerv1.ProviderConfig{}).
		Watches(&providerv1.ProviderConfigUsage{}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(r)
}
