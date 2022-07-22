package bucketcontroller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
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
			bucket := &cloudscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec:       cloudscalev1.BucketSpec{ForProvider: cloudscalev1.BucketParameters{BucketName: tc.givenNameInSpec}},
				Status:     cloudscalev1.BucketStatus{AtProvider: cloudscalev1.BucketObservation{BucketName: tc.givenNameInStatus}},
			}
			err := preventBucketRename(bucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}