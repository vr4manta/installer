package destroy

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/installer/pkg/asset/cluster/metadata"
	"github.com/openshift/installer/pkg/destroy/aws"
	"github.com/openshift/installer/pkg/destroy/providers"
	awstypes "github.com/openshift/installer/pkg/types/aws"
)

// New returns a Destroyer based on `metadata.json` in `rootDir`.
func New(logger logrus.FieldLogger, rootDir string) (providers.Destroyer, error) {
	clusterMetadata, err := metadata.Load(rootDir)
	if err != nil {
		return nil, err
	}

	platform := clusterMetadata.Platform()
	if platform == "" {
		return nil, errors.New("no platform configured in metadata")
	}

	// For AWS platforms, perform comprehensive permission validation upfront
	// This is similar to the permission checking done during cluster creation
	if platform == awstypes.Name {
		ctx := context.Background()

		// Check if cluster has dynamic dedicated hosts
		hasDynamicDHs, err := aws.HasDynamicDedicatedHosts(ctx, rootDir, logger)
		if err != nil {
			logger.WithError(err).Debug("Failed to check for dynamic dedicated hosts, continuing without dynamic DH permissions")
			hasDynamicDHs = false
		}
		if hasDynamicDHs {
			logger.Info("Dynamic dedicated hosts detected in cluster")
		}

		// Validate AWS permissions for destroy operation
		// This checks all required delete permissions plus dynamic DH permissions if needed
		if err := aws.ValidateDestroyPermissions(ctx, clusterMetadata, hasDynamicDHs, logger); err != nil {
			return nil, errors.Wrap(err, "AWS credential validation failed")
		}
	}

	creator, ok := providers.Registry[platform]
	if !ok {
		return nil, errors.Errorf("no destroyers registered for %q", platform)
	}
	return creator(logger, clusterMetadata)
}
