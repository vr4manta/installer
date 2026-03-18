package aws

import (
	"testing"

	awssession "github.com/openshift/installer/pkg/asset/installconfig/aws"
	"github.com/openshift/installer/pkg/types"
	awstypes "github.com/openshift/installer/pkg/types/aws"
)

func TestRequiredDestroyPermissionGroups(t *testing.T) {
	tests := []struct {
		name                     string
		metadata                 *types.ClusterMetadata
		hasDynamicDedicatedHosts bool
		expectedGroupsContain    []awssession.PermissionGroup
		expectedGroupsNotContain []awssession.PermissionGroup
	}{
		{
			name: "basic destroy without dynamic DHs",
			metadata: &types.ClusterMetadata{
				ClusterPlatformMetadata: types.ClusterPlatformMetadata{
					AWS: &awstypes.Metadata{
						Region: "us-east-1",
					},
				},
			},
			hasDynamicDedicatedHosts: false,
			expectedGroupsContain: []awssession.PermissionGroup{
				awssession.PermissionDeleteBase,
				awssession.PermissionDeleteNetworking,
				awssession.PermissionDeleteSharedNetworking,
				awssession.PermissionDedicatedHosts, // Always included for checking
			},
			expectedGroupsNotContain: []awssession.PermissionGroup{
				awssession.PermissionDynamicHostAllocation, // Not included without dynamic DHs
			},
		},
		{
			name: "destroy with dynamic dedicated hosts",
			metadata: &types.ClusterMetadata{
				ClusterPlatformMetadata: types.ClusterPlatformMetadata{
					AWS: &awstypes.Metadata{
						Region: "us-west-2",
					},
				},
			},
			hasDynamicDedicatedHosts: true,
			expectedGroupsContain: []awssession.PermissionGroup{
				awssession.PermissionDeleteBase,
				awssession.PermissionDeleteNetworking,
				awssession.PermissionDedicatedHosts,
				awssession.PermissionDynamicHostAllocation, // Included with dynamic DHs
			},
			expectedGroupsNotContain: []awssession.PermissionGroup{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := RequiredDestroyPermissionGroups(tt.metadata, tt.hasDynamicDedicatedHosts)

			// Check that expected groups are present
			groupsMap := make(map[awssession.PermissionGroup]bool)
			for _, g := range groups {
				groupsMap[g] = true
			}

			for _, expectedGroup := range tt.expectedGroupsContain {
				if !groupsMap[expectedGroup] {
					t.Errorf("Expected permission group %q to be included, but it was not", expectedGroup)
				}
			}

			for _, unexpectedGroup := range tt.expectedGroupsNotContain {
				if groupsMap[unexpectedGroup] {
					t.Errorf("Expected permission group %q to NOT be included, but it was", unexpectedGroup)
				}
			}
		})
	}
}
