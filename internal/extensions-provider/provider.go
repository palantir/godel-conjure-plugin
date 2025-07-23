package extensionsprovider

import (
	"maps"
	"os/exec"

	"github.com/palantir/pkg/safejson"
)

type ExtensionsProvider func(key string, tmpIRFilePath string, version string) (map[string]any, error)

func NewExtensionsProvider(url string, groupId string, assets []string) ExtensionsProvider {
	// url + "/artifactory/" + groupId + "/" + key is what is needed for resolvinng the older conjure IRs
	return func(conjureProject string, irFilePathWithoutExtensions string, version string) (map[string]any, error) {

		allExtensions := make(map[string]any)
		for _, asset := range assets {
			bytes, err := exec.Command(asset, "_assetInfo").Output()
			if err != nil {
				return nil, err
			}

			var response assetTypeResponse
			if err := safejson.Unmarshal(bytes, &response); err != nil {
				return nil, err
			}

			if response.Type != "conjure-ir-extensions-provider" { // skip assets that do not support `extensions`
				continue
			}

			arg, err := safejson.Marshal(extensionsAssetArgs{
				Proposed: irFilePathWithoutExtensions,
				Version:  version,
				Url:      url,
				GroupId:  groupId,
				Project:  conjureProject,
			})
			if err != nil {
				return nil, err
			}

			extensionBytes, err := exec.Command(asset, string(arg)).Output()
			if err != nil {
				return nil, err
			}

			var extensions map[string]any // must be this way for merging purposes
			if err := safejson.Unmarshal(extensionBytes, &extensions); err != nil {
				return nil, err
			}

			maps.Copy(allExtensions, extensions)
		}

		return allExtensions, nil
	}
}

type extensionsAssetArgs struct {
	Proposed string `json:"proposed,omitempty"` // proposed IR (copying naming from conjure backcompat)
	Version  string `json:"version,omitempty"`  // take this version if you incompatible
	Url      string `json:"url,omitempty"`
	GroupId  string `json:"group-id,omitempty"`
	Project  string `json:"project,omitempty"`
}

type assetTypeResponse struct {
	Type string `json:"type"`
}
