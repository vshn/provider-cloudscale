package bucketcontroller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BucketValidator validates admission requests.
type BucketValidator struct {
	log logr.Logger
}

// ValidateCreate implements admission.CustomValidator.
func (v *BucketValidator) ValidateCreate(_ context.Context, obj runtime.Object) error {
	bucket := obj.(*cloudscalev1.Bucket)
	v.log.V(1).Info("Validate create", "name", bucket.Name)
	// EndpointURL and Region are required by the API schema, no need to check.

	connectionSecretRef := bucket.Spec.WriteConnectionSecretToReference
	if connectionSecretRef == nil || connectionSecretRef.Name == "" || connectionSecretRef.Namespace == "" {
		return fmt.Errorf(".spec.writeConnectionSecretToReference name and namespace are required")
	}
	return nil
}

// ValidateUpdate implements admission.CustomValidator.
func (v *BucketValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) error {
	newBucket := newObj.(*cloudscalev1.Bucket)
	oldBucket := oldObj.(*cloudscalev1.Bucket)
	v.log.V(1).Info("Validate update")

	if oldBucket.Status.AtProvider.BucketName != "" {
		if newBucket.GetBucketName() != oldBucket.Status.AtProvider.BucketName {
			return fmt.Errorf("a bucket named %q has been created already, you cannot rename it",
				oldBucket.Status.AtProvider.BucketName)
		}
		if newBucket.Spec.ForProvider.Region != oldBucket.Spec.ForProvider.Region {
			return fmt.Errorf("a bucket named %q has been created already, you cannot change the region",
				oldBucket.Status.AtProvider.BucketName)
		}
	}
	newConnectionSecretRef := newBucket.Spec.WriteConnectionSecretToReference
	oldConnectionSecretRef := oldBucket.Spec.WriteConnectionSecretToReference
	if newConnectionSecretRef == nil || newConnectionSecretRef.Name != oldConnectionSecretRef.Name || newConnectionSecretRef.Namespace != oldConnectionSecretRef.Namespace {
		return fmt.Errorf(".spec.writeConnectionSecretToReference name and namespace cannot be changed")
	}
	return nil
}

// ValidateDelete implements admission.CustomValidator.
func (v *BucketValidator) ValidateDelete(_ context.Context, obj runtime.Object) error {
	res := obj.(*cloudscalev1.Bucket)
	v.log.V(1).Info("Validate delete (noop)", "name", res.Name)
	return nil
}
