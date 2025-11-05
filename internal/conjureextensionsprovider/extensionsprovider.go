package conjureextensionsprovider

type ConjureProjectConfig struct {
	ProjectName string `json:"projectName,omitempty"`
	GroupID     string `json:"groupId,omitempty"`
}

type Generator interface {
	GenerateConjureExtensions(projectConfig ConjureProjectConfig) (map[string]any, error)
}
