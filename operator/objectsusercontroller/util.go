package objectsusercontroller

import (
	cloudscalesdk "github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
)

func toTagMap(tags cloudscalev1.Tags) *cloudscalesdk.TagMap {
	if len(tags) == 0 {
		return nil
	}
	tagMap := make(cloudscalesdk.TagMap)
	for k, v := range tags {
		tagMap[k] = v
	}
	return &tagMap
}

func fromTagMap(tagMap cloudscalesdk.TagMap) cloudscalev1.Tags {
	tags := make(cloudscalev1.Tags)
	for k, v := range tagMap {
		tags[k] = v
	}
	return tags
}

func tagsNeedUpdate(desired cloudscalev1.Tags, observed cloudscalesdk.TagMap) bool {
	if len(observed) > 0 && len(desired) == 0 {
		// we have tags observed, but none are desired
		return true
	}
	// we have desired and observed tags, now compare each key-value pair
	for k, desiredValue := range desired {
		if observedValue, exists := observed[k]; exists {
			if observedValue != desiredValue {
				// a tag exists but it's not a desired value
				return true
			}
		} else {
			// a desired tag doesn't exist
			return true
		}
		continue
	}
	return false
}
