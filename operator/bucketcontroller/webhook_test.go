package bucketcontroller

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBucketValidator_ValidateCreate_RequireConnectionSecretRef(t *testing.T) {
	tests := map[string]struct {
		secretRef     *xpv1.SecretReference
		expectedError string
	}{
		"GivenWriteConnectionSecretToRef_ThenExpectNoError": {
			secretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		"GivenWriteConnectionSecretToRef_WhenNoName_ThenExpectError": {
			secretRef: &xpv1.SecretReference{
				Namespace: "namespace",
			},
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
		"GivenWriteConnectionSecretToRef_WhenNoNamespace_ThenExpectError": {
			secretRef: &xpv1.SecretReference{
				Name: "name",
			},
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
		"GivenWriteConnectionSecretToRef_WhenObjectIsNil_ThenExpectError": {
			secretRef:     nil,
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						// connection secret is being tested
						WriteConnectionSecretToReference: tc.secretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: cloudscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateCreate(nil, bucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_PreventBucketNameChange(t *testing.T) {
	tests := map[string]struct {
		newBucketName string
		oldBucketName string
		expectedError string
	}{
		"GivenNoNameInStatus_WhenNoNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "",
		},
		"GivenNoNameInStatus_WhenNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecSame_ThenExpectNil": {
			oldBucketName: "my-bucket",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecEmpty_ThenExpectNil": {
			oldBucketName: "bucket",
			newBucketName: "", // defaults to metadata.name
		},
		"GivenNameInStatus_WhenNameInSpecDifferent_ThenExpectError": {
			oldBucketName: "my-bucket",
			newBucketName: "different",
			expectedError: `a bucket named "my-bucket" has been created already, you cannot rename it`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"}},
					ForProvider:  cloudscalev1.BucketParameters{BucketName: tc.oldBucketName},
				},
				Status: cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: tc.oldBucketName}},
			}
			newBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"}},
					ForProvider:  cloudscalev1.BucketParameters{BucketName: tc.newBucketName},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_PreventRegionChange(t *testing.T) {
	tests := map[string]struct {
		newRegion     string
		oldRegion     string
		expectedError string
	}{
		"GivenRegionUnchanged_ThenExpectNil": {
			oldRegion: "region",
			newRegion: "region",
		},
		"GivenRegionChanged_ThenExpectError": {
			oldRegion:     "region",
			newRegion:     "different",
			expectedError: `a bucket named "bucket" has been created already, you cannot change the region`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"}},
					ForProvider:  cloudscalev1.BucketParameters{Region: tc.oldRegion}},
				Status: cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"}},
					ForProvider:  cloudscalev1.BucketParameters{Region: tc.newRegion}},
				Status: cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: "bucket"}},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_RequireConnectionSecretRef(t *testing.T) {
	tests := map[string]struct {
		newConnectionSecretRef *xpv1.SecretReference
		oldConnectionSecretRef *xpv1.SecretReference
		expectedError          string
	}{
		"GivenWriteConnectionSecretToRef_WhenOldIsEqualToNew_ThenExpectNoError": {
			newConnectionSecretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		"GivenWriteConnectionSecretToRef_WhenOldIsNotEqualToNew_ThenExpectError": {
			newConnectionSecretRef: &xpv1.SecretReference{
				Name:      "new-name",
				Namespace: "new-namespace",
			},
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "old-name",
				Namespace: "old-namespace",
			},
			expectedError: ".spec.writeConnectionSecretToReference name and namespace cannot be changed",
		},
		"GivenWriteConnectionSecretToRef_WhenNewIsNil_ThenExpectError": {
			newConnectionSecretRef: nil,
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "old-name",
				Namespace: "old-namespace",
			},
			expectedError: ".spec.writeConnectionSecretToReference name and namespace cannot be changed",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: tc.oldConnectionSecretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: cloudscalev1.BucketParameters{BucketName: "bucket"},
				},
				Status: cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: cloudscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: tc.newConnectionSecretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: cloudscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
