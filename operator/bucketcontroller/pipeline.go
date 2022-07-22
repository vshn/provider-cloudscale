package bucketcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProvisioningPipeline provisions Buckets using S3 client.
type ProvisioningPipeline struct {
	recorder event.Recorder
	kube     client.Client

	minio *minio.Client
}

// NewProvisioningPipeline returns a new instance of ProvisioningPipeline.
func NewProvisioningPipeline(kube client.Client, recorder event.Recorder, minio *minio.Client) *ProvisioningPipeline {
	return &ProvisioningPipeline{
		kube:     kube,
		recorder: recorder,
		minio:    minio,
	}
}

func fromManaged(mg resource.Managed) *cloudscalev1.Bucket {
	return mg.(*cloudscalev1.Bucket)
}
