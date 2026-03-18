package aws

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	awssession "github.com/openshift/installer/pkg/asset/installconfig/aws"
	"github.com/openshift/installer/pkg/types"
)

// RequiredDestroyPermissionGroups returns the permission groups required for destroying a cluster.
// This provides comprehensive upfront permission checking similar to the install-time validation.
func RequiredDestroyPermissionGroups(metadata *types.ClusterMetadata, hasDynamicDedicatedHosts bool) []awssession.PermissionGroup {
	// Start with base delete permissions that are always required
	permissionGroups := []awssession.PermissionGroup{
		awssession.PermissionDeleteBase,
	}

	// The metadata doesn't tell us everything about what was created, so we include
	// permissions for common destroy scenarios. This is conservative but ensures
	// users get clear upfront errors rather than mid-destroy failures.

	// Networking permissions - include both owned and shared scenarios
	// We include both because we can't easily determine from metadata alone
	// whether networking was BYO or created by installer
	permissionGroups = append(permissionGroups,
		awssession.PermissionDeleteNetworking,
		awssession.PermissionDeleteSharedNetworking,
	)

	// Check if cluster used dualstack based on region
	// Note: We could enhance this by storing more info in metadata, but for now
	// we include dualstack permissions for non-secret regions
	isSecretRegion, err := awssession.IsSecretRegion(metadata.ClusterPlatformMetadata.AWS.Region)
	if err == nil && !isSecretRegion {
		permissionGroups = append(permissionGroups, awssession.PermissionDeleteDualstackNetworking)
	}

	// Instance role/profile permissions - include both owned and shared scenarios
	permissionGroups = append(permissionGroups,
		awssession.PermissionDeleteSharedInstanceRole,
		awssession.PermissionDeleteSharedInstanceProfile,
	)

	// Hosted zone permissions
	permissionGroups = append(permissionGroups, awssession.PermissionDeleteHostedZone)

	// Ignition object deletion permissions
	permissionGroups = append(permissionGroups, awssession.PermissionDeleteIgnitionObjects)

	// Dedicated host permissions - always include DescribeHosts since we need to check
	permissionGroups = append(permissionGroups, awssession.PermissionDedicatedHosts)

	// If dynamic dedicated hosts were detected, add allocation/release permissions
	if hasDynamicDedicatedHosts {
		permissionGroups = append(permissionGroups, awssession.PermissionDynamicHostAllocation)
	}

	return permissionGroups
}

// ValidateDestroyPermissions validates that AWS credentials have all necessary permissions
// to destroy the cluster. This provides comprehensive upfront validation similar to install-time checks.
func ValidateDestroyPermissions(ctx context.Context, metadata *types.ClusterMetadata, hasDynamicDedicatedHosts bool, logger logrus.FieldLogger) error {
	logger.Info("Validating AWS permissions for cluster destroy")

	region := metadata.ClusterPlatformMetadata.AWS.Region
	endpoints := metadata.AWS.ServiceEndpoints

	// Get required permission groups for this destroy operation
	requiredGroups := RequiredDestroyPermissionGroups(metadata, hasDynamicDedicatedHosts)

	logger.Debugf("Checking %d permission groups for destroy", len(requiredGroups))

	// Create AWS config
	awsConfig, err := awssession.GetConfigWithOptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AWS config: %w", err)
	}

	// Build IAM endpoint URL
	iamEndpoint := ""
	for _, endpoint := range endpoints {
		if endpoint.Name == "iam" {
			iamEndpoint = endpoint.URL
			break
		}
	}

	// Validate credentials using the same CCO logic as install-time
	if err := awssession.ValidateCreds(ctx, awsConfig, requiredGroups, region, iamEndpoint); err != nil {
		// Provide helpful error message with permission groups that failed
		return fmt.Errorf("AWS credentials insufficient for cluster destroy. "+
			"Please ensure your credentials have permissions for the following groups: %v. "+
			"Validation error: %w", requiredGroups, err)
	}

	logger.Info("AWS credentials validated successfully for cluster destroy")
	return nil
}
