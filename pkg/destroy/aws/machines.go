package aws

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	awssession "github.com/openshift/installer/pkg/asset/installconfig/aws"
	"github.com/openshift/installer/pkg/types"
)

// HasDynamicDedicatedHosts checks if the cluster has any machines configured with dynamic dedicated host allocation.
// It reads the kubeconfig from the installer directory and queries the cluster for Machine CRs in the
// openshift-machine-api namespace to determine if any are using dedicated hosts.
func HasDynamicDedicatedHosts(ctx context.Context, rootDir string, logger logrus.FieldLogger) (bool, error) {
	kubeconfigPath := filepath.Join(rootDir, "auth", "kubeconfig")

	// Check if kubeconfig exists
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		logger.Debugf("Kubeconfig not found at %s, skipping dynamic dedicated host check", kubeconfigPath)
		return false, nil
	}

	// Load kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		logger.WithError(err).Debug("Failed to load kubeconfig, skipping dynamic dedicated host check")
		return false, nil
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Debug("Failed to create Kubernetes client, skipping dynamic dedicated host check")
		return false, nil
	}

	// Define the GVR for Machine CRs
	machineGVR := schema.GroupVersionResource{
		Group:    "machine.openshift.io",
		Version:  "v1beta1",
		Resource: "machines",
	}

	// List all machines in the openshift-machine-api namespace
	machines, err := dynamicClient.Resource(machineGVR).Namespace("openshift-machine-api").List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.WithError(err).Debug("Failed to list machines, skipping dynamic dedicated host check")
		return false, nil
	}

	// Check each machine for dedicated host configuration
	for _, machine := range machines.Items {
		if hasDedicatedHost(&machine) {
			logger.Debug("Found machine with dedicated host allocation configured")
			return true, nil
		}
	}

	logger.Debug("No machines with dedicated host allocation found")
	return false, nil
}

// hasDedicatedHost checks if a Machine CR has dedicated host allocation configured.
// It looks for the placement.host configuration in the provider spec.
func hasDedicatedHost(machine *unstructured.Unstructured) bool {
	// Get the providerSpec from the machine
	providerSpec, found, err := unstructured.NestedMap(machine.Object, "spec", "providerSpec", "value")
	if err != nil || !found {
		return false
	}

	// Check for placement.host configuration
	placement, found, err := unstructured.NestedMap(providerSpec, "placement")
	if err != nil || !found {
		return false
	}

	// Check if host field exists in placement
	_, found, err = unstructured.NestedMap(placement, "host")
	if err != nil {
		return false
	}

	return found
}

// ValidateDedicatedHostPermissions validates that the user has the necessary permissions
// to release dedicated hosts during cluster destruction.
func ValidateDedicatedHostPermissions(ctx context.Context, metadata *types.ClusterMetadata, logger logrus.FieldLogger) error {
	logger.Info("Validating permissions for dynamic dedicated host cleanup")

	region := metadata.ClusterPlatformMetadata.AWS.Region
	endpoints := metadata.AWS.ServiceEndpoints

	// Create AWS config with region
	awsConfig, err := awssession.GetConfigWithOptions(ctx, config.WithRegion(region))
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

	// Validate that the user has permissions to allocate and release dedicated hosts
	requiredPermissions := []string{
		"ec2:DescribeHosts",
		"ec2:AllocateHosts",
		"ec2:ReleaseHosts",
	}

	// Use the same CCO validation logic used during install
	if err := awssession.ValidateCreds(ctx, awsConfig, []awssession.PermissionGroup{
		awssession.PermissionDedicatedHosts,
		awssession.PermissionDynamicHostAllocation,
	}, region, iamEndpoint); err != nil {
		return fmt.Errorf("AWS credentials lack required permissions for dedicated host cleanup. Required permissions: %v. Error: %w",
			requiredPermissions, err)
	}

	logger.Info("AWS credentials validated successfully for dedicated host cleanup")
	return nil
}
