package bucketcontroller

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

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
