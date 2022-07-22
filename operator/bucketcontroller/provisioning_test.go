package bucketcontroller

import (
	"context"
	"testing"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_preventBucketRename(t *testing.T) {
	tests := map[string]struct {
		givenNameInStatus string
		givenNameInSpec   string
		expectedError     string
	}{
		"GivenNoNameInStatus_WhenNoNameInSpec_ThenExpectNil": {
			givenNameInStatus: "",
			givenNameInSpec:   "",
		},
		"GivenNoNameInStatus_WhenNameInSpec_ThenExpectNil": {
			givenNameInStatus: "",
			givenNameInSpec:   "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecSame_ThenExpectNil": {
			givenNameInStatus: "my-bucket",
			givenNameInSpec:   "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecEmpty_ThenExpectNil": {
			givenNameInStatus: "bucket",
			givenNameInSpec:   "", // defaults to metadata.name
		},
		"GivenNameInStatus_WhenNameInSpecDifferent_ThenExpectError": {
			givenNameInStatus: "my-bucket",
			givenNameInSpec:   "different",
			expectedError:     `a bucket named "my-bucket" has been previously created, you cannot rename it. Either revert 'spec.bucketName' back to "my-bucket" or delete the bucket and recreate using a new name`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := pipeline.MutableContext(context.Background())

			bucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{BucketName: tc.givenNameInSpec}},
				Status:     cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: tc.givenNameInStatus}},
			}
			pipeline.StoreInContext(ctx, BucketKey{}, bucket)

			err := preventBucketRename(ctx)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_validateSecret(t *testing.T) {
	tests := map[string]struct {
		givenSecretData map[string][]byte
		expectedError   string
	}{
		"GivenExpectedKeys_ThenExpectNil": {
			givenSecretData: map[string][]byte{
				cloudscalev1.AccessKeyIDName:     []byte("a"),
				cloudscalev1.SecretAccessKeyName: []byte("s"),
			},
		},
		"GivenMissingAccessKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.SecretAccessKeyName: []byte("s"),
			},
			expectedError: `secret "secret" is missing on of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenMissingSecretKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.AccessKeyIDName: []byte("a"),
			},
			expectedError: `secret "secret" is missing on of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenEmptyAccessKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.AccessKeyIDName:     []byte(""),
				cloudscalev1.SecretAccessKeyName: []byte("s"),
			},
			expectedError: `secret "secret" is missing on of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenEmptySecretKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.SecretAccessKeyName: []byte(""),
				cloudscalev1.AccessKeyIDName:     []byte("a"),
			},
			expectedError: `secret "secret" is missing on of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenNilData_ThenExpectError": {
			givenSecretData: nil,
			expectedError:   `secret "secret" does not have any data`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := pipeline.MutableContext(context.Background())
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret"},
				Data:       tc.givenSecretData}
			pipeline.StoreInContext(ctx, CredentialsSecretKey{}, secret)

			err := validateSecret(ctx)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
