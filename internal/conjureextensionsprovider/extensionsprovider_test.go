package conjureextensionsprovider_test

import (
	"os"

	"github.com/palantir/godel-conjure-plugin/v6/internal/conjureextensionsprovider"
	"github.com/palantir/pkg/cobracli"
)

// This file contains an example of how a project may implement an asset using the provided API.

// not actually a main in this test file, but example of what it would look like
func main() {
	rootCmd := conjureextensionsprovider.NewGeneratorRootCommand(
		"recommended-products-extensions-provider",
		"Generates the \"recommended-products\" Conjure extensions based on project configuration",
		NewRecommendedProductsExtensionsProvider(),
	)
	os.Exit(cobracli.ExecuteWithDefaultParams(rootCmd))
}

// This would likely be in a separate package (non-main package) in a real project

var _ conjureextensionsprovider.Generator = (*recommendedProductsExtensionsProvider)(nil)

type recommendedProductsExtensionsProvider struct {
}

func NewRecommendedProductsExtensionsProvider() conjureextensionsprovider.Generator {
	return &recommendedProductsExtensionsProvider{}
}

func (r *recommendedProductsExtensionsProvider) GenerateConjureExtensions(projectConfig conjureextensionsprovider.ConjureProjectConfig) (map[string]any, error) {
	// not real or valid, just an example/placeholder
	return map[string]any{
		"recommended-products": []string{"ProductA", "ProductB", "ProductC"},
	}, nil
}
