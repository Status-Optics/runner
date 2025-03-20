package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	// "context"
	// "encoding/json"
	// "github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	// "github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/aranw/yamlcfg"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-co-op/gocron"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Config struct {
	Runner struct {
		Name     string `yaml:"name"`
		Version  string `yaml:"version"`
		LogLevel string `yaml:"log_level"`
	} `yaml:"runner"`
	Tests     []TestConfig `yaml:"tests"`
	Reporting struct {
		Endpoints []struct {
			Type   string `yaml:"type"`
			Format string `yaml:"format,omitempty"`
		} `yaml:"endpoints"`
	} `yaml:"reporting"`
}

type TestConfig struct {
	Name   string `yaml:"name"`
	Source struct {
		Repository string `yaml:"repository"`
		Branch     string `yaml:"branch"`
		Directory  string `yaml:"directory"`
		GetUser    string `yaml:"get_user"`  // This will be an environment variable that should be substituted
		GetToken   string `yaml:"get_token"` // This will be an environment variable that should be substituted
	} `yaml:"source"`
	Language   string `yaml:"language"`
	SetupCmd   string `yaml:"setup_cmd"`
	Executable string `yaml:"executable"`
	Command    string `yaml:"command"`
	Frequency  int32  `yaml:"frequency"`
}

func createLogger(logLevel string) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	newLogLevel := zap.InfoLevel
	switch logLevel {
	case "debug":
		newLogLevel = zap.DebugLevel
	case "info":
		newLogLevel = zap.InfoLevel
	case "warn":
		newLogLevel = zap.WarnLevel
	case "error":
		newLogLevel = zap.ErrorLevel
	case "fatal":
		newLogLevel = zap.FatalLevel
	case "panic":
		newLogLevel = zap.PanicLevel
	}

	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(newLogLevel),
		Development:       false,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "json",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
		InitialFields: map[string]interface{}{
			"pid": os.Getpid(),
		},
	}

	return zap.Must(config.Build())
}

func main() {
	// Bootstrap the application
	config, err := bootstrap()
	if err != nil {
		fmt.Println("Error bootstrapping application:", err)
		os.Exit(1)
	}

	// Set up logger
	logger := createLogger(config.Runner.LogLevel)
	defer logger.Sync()
	logger.Info("Starting Runner")

	// define a var to hold the tests
	testConfigs := []TestConfig{}
	testConfigs = append(testConfigs, config.Tests...)

	// Schedule configuration updates
	s := gocron.NewScheduler(time.UTC)

	// Add the tests to the scheduler
	for _, testConfig := range testConfigs {
		s.Every(time.Duration(testConfig.Frequency) * time.Second).Do(func() {
			// Pull the latest code from the repository
			repo, err := cloneOrUpdateRepo(testConfig.Source.Repository, testConfig.Source.GetUser, testConfig.Source.GetToken, testConfig.Name)
			if err != nil {
				logger.Error("Error cloning/updating test repository", zap.String("repository", testConfig.Source.Repository), zap.Error(err))
				return
			}

			// If Language is python, create a virtual environment and install dependencies
			if testConfig.Language == "python" {
				// Install dependencies
				cmd := exec.Command("pip3", "install", "-r", "/tmp/"+testConfig.Name+"/"+testConfig.Source.Directory+"/requirements.txt")
				err = cmd.Run()
				if err != nil {
					logger.Error("Error installing dependencies", zap.Error(err))
					return
				}
				logger.Info("Dependencies installed", zap.String("test", testConfig.Name))

			}

			// Execute the test command
			dirPath := "/tmp/" + testConfig.Name
			if testConfig.Source.Directory != "" {
				dirPath += "/" + testConfig.Source.Directory
			}
			runString := fmt.Sprintf("cd %s && %s", dirPath, testConfig.Executable+" "+testConfig.Command)
			fmt.Println(runString)
			cmd := exec.Command("bash", "-c", runString)

			// Get the commit hash from repo
			ref, _ := repo.Head()
			commit, err := repo.CommitObject(ref.Hash())
			if err != nil {
				logger.Error("Error getting commit object", zap.Error(err))
				return
			}

			cmd.Env = append(os.Environ(), fmt.Sprintf("TEST_GIT_COMMIT=%s", commit.Hash.String()))

			// Print the output of the command
			output, err := cmd.Output()
			if err != nil {
				logger.Error("Error executing test command", zap.Error(err))
				return
			}
			// Log the output
			logger.Debug("Test executed", zap.String("test", testConfig.Name))
			logger.Debug("Output", zap.String("output", string(output)))
			for _, endpoint := range config.Reporting.Endpoints {
				if endpoint.Type == "stdout" {
					logToStdout(string(output), endpoint.Format)
				}
			}
			logger.Debug("Test completed", zap.String("test", testConfig.Name))
		})
		logger.Debug("Scheduled test", zap.String("test", testConfig.Name), zap.Int32("frequency", testConfig.Frequency))
	}

	s.StartBlocking()
}

func cloneOrUpdateRepo(repoURL string, user string, token string, dirName string) (*git.Repository, error) {

	repo := &git.Repository{}
	var err error
	// Check if the directory exists
	if _, err = os.Stat("/tmp/" + dirName); err == nil {
		// Open the existing repository
		repo, err = git.PlainOpen("/tmp/" + dirName)
		if err != nil {
			fmt.Println("Error opening repository:", err)
			return nil, err
		}

		// Pull the latest changes
		w, err := repo.Worktree()
		if err != nil {
			fmt.Println("Error getting worktree:", err)
			return nil, err
		}
		err = w.Pull(&git.PullOptions{
			Auth: &http.BasicAuth{
				Username: user,
				Password: token,
			},
		})
		if err != nil {
			if err.Error() != "already up-to-date" {
				fmt.Println("Error pulling changes:", err)
				return nil, err
			} else {
				fmt.Println("Repository is already up-to-date")
			}
		} else {
			fmt.Println("Successfully pulled latest changes")
		}
	} else if os.IsNotExist(err) {
		// If the directory does not exist, clone the repository
		repo, err = git.PlainClone("/tmp/"+dirName, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
			Auth: &http.BasicAuth{
				Username: user,
				Password: token,
			},
		})
		if err != nil {
			fmt.Println("Error cloning repository:", err)
			return nil, err
		}
		fmt.Println(repo)
	}
	return repo, nil
}

func bootstrap() (Config, error) {
	// Retrieve bootstrapping environment variables
	configRepo := os.Getenv("RUNNER_CONFIG_REPO")
	configBranch := os.Getenv("RUNNER_CONFIG_BRANCH")
	if configBranch == "" {
		configBranch = "main"
	}
	configPath := os.Getenv("RUNNER_CONFIG_PATH")
	if configPath == "" {
		configPath = "statusoptics.yaml"
	}

	gitConfigToken := os.Getenv("RUNNER_GIT_CONFIG_TOKEN")
	if gitConfigToken == "" {
		// Error out if the token is not provided
		fmt.Println("Error: RUNNER_GIT_CONFIG_TOKEN environment variable is required")
		return Config{}, fmt.Errorf("RUNNER_GIT_CONFIG_TOKEN environment variable is required")
	}

	gitConfigUser := os.Getenv("RUNNER_GIT_CONFIG_USER")
	if gitConfigUser == "" {
		// Error out if the user is not provided
		fmt.Println("Error: RUNNER_GIT_CONFIG_USER environment variable is required")
		return Config{}, fmt.Errorf("RUNNER_GIT_CONFIG_USER environment variable is required")
	}

	// Clone or update the config repository
	_, err := cloneOrUpdateRepo(configRepo, gitConfigUser, gitConfigToken, "runner-config")
	if err != nil {
		fmt.Println("Error cloning/updating config repository:", err)
		return Config{}, err
	}

	// Read and parse runner.yaml
	config, err := readAndParseConfig("/tmp/runner-config/" + configPath)
	if err != nil {
		fmt.Println("Error reading/parsing config:", err)
		return Config{}, err
	}
	// fmt.Println(config)
	return config, nil
}

func readAndParseConfig(filePath string) (Config, error) {
	// read the file, unmarshal the yaml, and return the result.

	cfg, err := yamlcfg.Parse[Config](filePath)
	if err != nil {
		return Config{}, err
	}

	logLevel := os.Getenv("RUNNER_LOG_LEVEL")
	if logLevel != "" {
		cfg.Runner.LogLevel = logLevel
	}
	// If log level is not set, default to "info"
	if cfg.Runner.LogLevel == "" {
		cfg.Runner.LogLevel = "info"
	}

	fmt.Println("Log level:", cfg.Runner.LogLevel)
	// print the parsed config for debugging
	// fmt.Printf("Parsed config: %+v\n", cfg)

	return *cfg, nil

}

func logToStdout(data string, format string) {
	if format == "json" {
		fmt.Println(data)
	} else {
		fmt.Println(data)
	}
}
