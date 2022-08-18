package bucketcontroller

import (
	"context"
	"fmt"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EndpointName is the endpoint field name in credential secret
	EndpointName = "ENDPOINT"
	// RegionName is the region field name in credential secret
	RegionName = "REGION"
	// BucketName is the bucket field name in credential secret
	BucketName = "BUCKET_NAME"
)

// ProvisioningPipeline provisions Buckets using S3 client.
type ProvisioningPipeline struct {
	recorder event.Recorder
	kube     client.Client

	minio *minio.Client
}

type pipelineContext struct {
	context.Context
	Bucket                      *cloudscalev1.Bucket
	ObjectsUserCredentialSecret *corev1.Secret
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

func (p *ProvisioningPipeline) fetchSecret(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	bucket := ctx.Bucket

	secret := &corev1.Secret{}
	secretRef := bucket.Spec.ForProvider.CredentialsSecretRef
	err := p.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, secret)
	if err != nil {
		return err
	}
	ctx.ObjectsUserCredentialSecret = secret
	log.V(1).Info("Fetched credentials secret", "secret name", fmt.Sprintf("%s/%s", secretRef.Namespace, secretRef.Name))
	return nil
}
