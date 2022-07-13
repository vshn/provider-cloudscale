package operator

import (
	"github.com/vshn/appcat-service-s3/operator/bucketcontroller"
	"github.com/vshn/appcat-service-s3/operator/cloudscale"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupControllers creates all controllers with the supplied logger and adds them to the supplied manager.
func SetupControllers(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		cloudscale.SetupController,
		bucketcontroller.SetupController,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}