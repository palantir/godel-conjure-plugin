package conjureextensionsprovider

import (
	"encoding/json"
	"os/exec"

	"github.com/palantir/godel-conjure-plugin/v6/internal/assetapi"
	"github.com/palantir/godel-conjure-plugin/v6/internal/cmdutils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewAssetProvider(assetPath string) Generator {
	return &assetExtensionsProvider{
		assetPath: assetPath,
	}
}

type assetExtensionsProvider struct {
	assetPath string
}

var _ Generator = (*assetExtensionsProvider)(nil)

func (c *assetExtensionsProvider) GenerateConjureExtensions(projectConfig ConjureProjectConfig) (map[string]any, error) {
	projectConfigJSON, err := json.Marshal(projectConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal Conjure project config as JSON")
	}
	cmd := exec.Command(c.assetPath,
		generateConjureExtensionsCommand,
		"--"+conjureProjectConfigJSONFlagName, string(projectConfigJSON),
	)
	var output map[string]any
	output, err = cmdutils.GetCommandOutputAsJSON[map[string]any](cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate conjure extensions plugin asset")
	}
	return output, nil
}

// NewGeneratorRootCommand creates a Cobra root command for an asset that provides Conjure extensions.
// The returned command has the required subcommands hooked up for reporting the asset type and generating Conjure
// extensions according to the asset spec using the provided generator as its implementation.
func NewGeneratorRootCommand(name, description string, generator Generator) *cobra.Command {
	rootCmd := assetapi.AssetRootCmd(assetapi.ConjureExtensionsProvider, name, description)

	rootCmd.AddCommand(newGenerateConjureExtensionsCmd(generator))

	return rootCmd
}

const (
	generateConjureExtensionsCommand = "generate-conjure-extensions"

	conjureProjectConfigJSONFlagName = "conjure-project-config-json"
)

func newGenerateConjureExtensionsCmd(generator Generator) *cobra.Command {
	var (
		conjureProjectConfigJSON string
	)
	generateConjureExtensionsCmd := &cobra.Command{
		Use:   generateConjureExtensionsCommand,
		Short: "Generate Conjure extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			generatedExtensions, err := generator.GenerateConjureExtensions(ConjureProjectConfig{})
			if err != nil {
				return err
			}
			outputJSON, err := json.Marshal(generatedExtensions)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal output as JSON")
			}
			cmd.Print(string(outputJSON))
			return nil
		},
	}
	generateConjureExtensionsCmd.Flags().StringVar(&conjureProjectConfigJSON, conjureProjectConfigJSONFlagName, "", "Conjure project configuration encoded as JSON")
	mustMarkFlagsRequired(generateConjureExtensionsCmd, conjureProjectConfigJSONFlagName)
	return generateConjureExtensionsCmd
}

func mustMarkFlagsRequired(cmd *cobra.Command, flagNames ...string) {
	for _, currFlagName := range flagNames {
		if err := cmd.MarkFlagRequired(currFlagName); err != nil {
			panic(err)
		}
	}
}
