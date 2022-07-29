package operator

import (
	"github.com/vshn/provider-cloudscale/operator/bucketcontroller"
	"github.com/vshn/provider-cloudscale/operator/configcontroller"
	"github.com/vshn/provider-cloudscale/operator/objectsusercontroller"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupControllers creates all controllers and adds them to the supplied manager.
func SetupControllers(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		objectsusercontroller.SetupController,
		bucketcontroller.SetupController,
		configcontroller.SetupController,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhooks creates all webhooks and adds them to the supplied manager.
func SetupWebhooks(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		bucketcontroller.SetupWebhook,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
