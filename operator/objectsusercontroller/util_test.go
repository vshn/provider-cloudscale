package objectsusercontroller

import (
	"testing"

	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/stretchr/testify/assert"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
)

func TestTagsNeedUpdate(t *testing.T) {
	tests := map[string]struct {
		desiredTags    cloudscalev1.Tags
		observedTags   cloudscalesdk.TagMap
		expectedResult bool
	}{
		"GivenNilDesiredTags_WhenObservedTagsNil_ThenExpectFalse":     {desiredTags: nil, observedTags: nil, expectedResult: false},
		"GivenEmptyDesiredTags_WhenObservedTagsEmpty_ThenExpectFalse": {desiredTags: map[string]string{}, observedTags: map[string]string{}, expectedResult: false},
		"GivenDesiredTags_WhenObservedTagsEmpty_ThenExpectTrue":       {desiredTags: map[string]string{"desired": "yes"}, observedTags: map[string]string{}, expectedResult: true},
		"GivenEmptyDesiredTags_WhenObservedTagsGiven_ThenExpectTrue":  {desiredTags: map[string]string{}, observedTags: map[string]string{"observed": "yes"}, expectedResult: true},
		"GivenMultipleDesiredTags_WhenObservedTagsEqual_ThenExpectFalse": {
			desiredTags:    map[string]string{"tag1": "value1", "tag2": "value2"},
			observedTags:   map[string]string{"tag1": "value1", "tag2": "value2"},
			expectedResult: false,
		},
		"GivenMultipleDesiredTags_WhenObservedTagsMissing_ThenExpectTrue": {
			desiredTags:    map[string]string{"tag1": "value1", "tag2": "value2"},
			observedTags:   map[string]string{"tag1": "value1"},
			expectedResult: true,
		},
		"GivenMultipleDesiredTags_WhenObservedTagsWrongValue_ThenExpectTrue": {
			desiredTags:    map[string]string{"tag1": "value1"},
			observedTags:   map[string]string{"tag1": "another"},
			expectedResult: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tagsNeedUpdate(tc.desiredTags, tc.observedTags)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
