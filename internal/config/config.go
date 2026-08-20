package config

// Config holds the complete application configuration loaded from environment variables.
type Config struct {
	LogLevel        string
	DataDir         string
	ProjectsDirPath string
	Embeddings      Embeddings
	Memory          Memory
	Web             Web
}

// Embeddings configures the embedding provider.
type Embeddings struct {
	Provider  string
	URL       string
	Model     string
	APIKey    string
	BatchSize int
}

// Memory configures chat/dialogue memory providers.
type Memory struct {
	Cascade ProviderConfig
	Cursor  ProviderConfig
}

// ProviderConfig configures a single memory provider (cascade or cursor).
type ProviderConfig struct {
	Enabled              bool
	SourceDirs           []string
	MinUserMessageLength int
}

// Web configures the optional HTTP dashboard server.
type Web struct {
	Enabled bool
	Port    int
}

// Directory configures a single indexed project.
type Directory struct {
	Path          string   `json:"path"`
	Name          string   `json:"name"`
	Extensions    []string `json:"extensions,omitempty"`
	Include       []string `json:"include,omitempty"`
	Exclude       []string `json:"exclude,omitempty"`
	MaxFileSizeMB int      `json:"max_file_size_mb,omitempty"`
}

const (
	defaultLogLevel             = "info"
	defaultDataDir              = "${HOME}/.gnostis/data"
	DefaultProjectsDir          = "${HOME}/.gnostis/projects"
	DefaultMemoryDataDir        = "${HOME}/.gnostis/data/memory"
	defaultProvider             = "ollama"
	defaultURL                  = "http://localhost:11434/v1"
	defaultModel                = "nomic-embed-text"
	defaultBatchSize            = 32
	defaultWebEnabled           = true
	defaultWebPort              = 7878
	defaultMinUserMessageLength = 10
)
