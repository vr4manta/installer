package openshift

import (
	"context"
	"os"
	"path/filepath"

	"github.com/openshift/installer/pkg/asset"
	"github.com/openshift/installer/pkg/asset/templates/content"
)

const (
	cloudCredsSecretFileName = "cloud-creds-secret.yaml.template"
)

var _ asset.WritableAsset = (*CloudCredsSecret)(nil)

// CloudCredsSecret is the constant to represent contents of corresponding yaml file
type CloudCredsSecret struct {
	FileList []*asset.File
}

// Dependencies returns all of the dependencies directly needed by the asset
func (t *CloudCredsSecret) Dependencies() []asset.Asset {
	return []asset.Asset{}
}

// Name returns the human-friendly name of the asset.
func (t *CloudCredsSecret) Name() string {
	return "CloudCredsSecret"
}

// Generate generates the actual files by this asset
func (t *CloudCredsSecret) Generate(_ context.Context, parents asset.Parents) error {
	fileName := cloudCredsSecretFileName
	data, err := content.GetOpenshiftTemplate(fileName)
	if err != nil {
		return err
	}
	t.FileList = []*asset.File{
		{
			Filename: filepath.Join(content.TemplateDir, fileName),
			Data:     []byte(data),
		},
	}
	return nil
}

// Files returns the files generated by the asset.
func (t *CloudCredsSecret) Files() []*asset.File {
	return t.FileList
}

// Load returns the asset from disk.
func (t *CloudCredsSecret) Load(f asset.FileFetcher) (bool, error) {
	file, err := f.FetchByName(filepath.Join(content.TemplateDir, cloudCredsSecretFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	t.FileList = []*asset.File{file}
	return true, nil
}
