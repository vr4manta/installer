package aws

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
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
