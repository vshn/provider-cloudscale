package bucketcontroller

import (
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_bucketConnector_validateSecret(t *testing.T) {
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
			expectedError: `secret "default/secret" is missing one of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenMissingSecretKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.AccessKeyIDName: []byte("a"),
			},
			expectedError: `secret "default/secret" is missing one of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenEmptyAccessKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.AccessKeyIDName:     []byte(""),
				cloudscalev1.SecretAccessKeyName: []byte("s"),
			},
			expectedError: `secret "default/secret" is missing one of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenEmptySecretKey_ThenExpectError": {
			givenSecretData: map[string][]byte{
				cloudscalev1.SecretAccessKeyName: []byte(""),
				cloudscalev1.AccessKeyIDName:     []byte("a"),
			},
			expectedError: `secret "default/secret" is missing one of the following keys or content: [AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY]`,
		},
		"GivenNilData_ThenExpectError": {
			givenSecretData: nil,
			expectedError:   `secret "default/secret" does not have any data`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "default"},
				Data:       tc.givenSecretData}

			c := &bucketConnector{}
			ctx := &connectContext{Context: context.Background(), credentialsSecret: secret}
			err := c.validateSecret(ctx)

			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_isTLSEnabled(t *testing.T) {
	tests := map[string]struct {
		givenURL     *url.URL
		expectedBool bool
	}{
		"GivenNoScheme_ThenExpectTrue": {
			givenURL:     mustParse("domain.tld"),
			expectedBool: true,
		},
		"GivenHttpScheme_WhenLowercase_ThenExpectFalse": {
			givenURL:     mustParse("http://domain.tld"),
			expectedBool: false,
		},
		"GivenHttpScheme_WhenUppercase_ThenExpectFalse": {
			givenURL:     mustParse("HTTP://domain.tld"),
			expectedBool: false,
		},
		"GivenHttpsScheme_WhenUppercase_ThenExpectTrue": {
			givenURL:     mustParse("HTTPS://domain.tld"),
			expectedBool: true,
		},
		"GivenHttpsScheme_ThenLowercase_ExpectTrue": {
			givenURL:     mustParse("https://domain.tld"),
			expectedBool: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := isTLSEnabled(tc.givenURL)
			require.Equal(t, tc.expectedBool, result)
		})
	}
}

func mustParse(rawUrl string) *url.URL {
	u, err := url.Parse(rawUrl)
	if err != nil {
		panic(err)
	}
	return u
}
