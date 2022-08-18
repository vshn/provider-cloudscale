package bucketcontroller

import (
	"context"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *ProvisioningPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	s3Client := p.minio
	bucket := fromManaged(mg)

	bucketName := bucket.GetBucketName()
	exists, err := s3Client.BucketExists(ctx, bucketName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot determine whether bucket exists")
	}
	if exists {
		bucket.Status.AtProvider.BucketName = bucketName
		bucket.SetConditions(xpv1.Available())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: toConnectionDetails(bucket)}, nil
	}
	return managed.ExternalObservation{}, nil
}

func toConnectionDetails(bucket *cloudscalev1.Bucket) managed.ConnectionDetails {
	return map[string][]byte{
		EndpointName: []byte(bucket.Spec.ForProvider.EndpointURL),
		RegionName:   []byte(bucket.Spec.ForProvider.Region),
		BucketName:   []byte(bucket.Spec.ForProvider.BucketName),
	}
}
