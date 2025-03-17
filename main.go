package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/aranw/yamlcfg"

	"github.com/go-co-op/gocron"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Config struct {
	Runner struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"runner"`
	Tests []TestConfig `yaml:"tests"`
}

type TestConfig struct {
	Name   string `yaml:"name"`
	Source struct {
		Repository string `yaml:"repository"`
		Branch     string `yaml:"branch"`
		GetUser    string `yaml:"get_user"`  // This will be an environment variable that should be substituted
		GetToken   string `yaml:"get_token"` // This will be an environment variable that should be substituted
	} `yaml:"source"`
	Language   string `yaml:"language"`
	WorkingDir string `yaml:"working_dir"`
	SetupCmd   string `yaml:"setup_cmd"`
	Executable string `yaml:"executable"`
	Command    string `yaml:"command"`
	Frequency  int32  `yaml:"frequency"`
}

func main() {
	// Bootstrap the application
	config, err := bootstrap()
	if err != nil {
		fmt.Println("Error bootstrapping application:", err)
		os.Exit(1)
	}

	// define a var to hold the tests
	testConfigs := []TestConfig{}
	for _, test := range config.Tests {
		testConfigs = append(testConfigs, test)
	}

	// Schedule configuration updates
	s := gocron.NewScheduler(time.UTC)

	// Add the tests to the scheduler
	for _, testConfig := range testConfigs {
		s.Every(time.Duration(testConfig.Frequency) * time.Second).Do(func() {
			// Pull the latest code from the repository
			err := cloneOrUpdateRepo(testConfig.Source.Repository, testConfig.Source.GetUser, testConfig.Source.GetToken, testConfig.Name)
			if err != nil {
				fmt.Println("Error cloning/updating test repository:", err)
				return
			}

			// If Language is python, create a virtual environment and install dependencies
			if testConfig.Language == "python" {
				// Create a virtual environment
				cmd := exec.Command("python3", "-m", "venv", "/tmp/"+testConfig.Name+"/"+testConfig.WorkingDir+"/venv")
				err := cmd.Run()
				if err != nil {
					fmt.Printf("Error creating virtual environment: %s\n", err)
					return
				}
				// Install dependencies
				cmd = exec.Command("/tmp/"+testConfig.Name+"/"+testConfig.WorkingDir+"/venv/bin/pip", "install", "-r", "/tmp/"+testConfig.Name+"/"+testConfig.WorkingDir+"/requirements.txt")
				err = cmd.Run()
				if err != nil {
					fmt.Printf("Error installing dependencies: %s\n", err)
					return
				}
				fmt.Printf("Virtual environment created and dependencies installed for test: %s\n", testConfig.Name)

				// Activate the virtual environment
				// cmd = exec.Command("source", "/tmp/"+testConfig.Name+"/"+testConfig.WorkingDir+"/venv/bin/activate")
				// err = cmd.Run()
				// if err != nil {
				// 	fmt.Printf("Error activating virtual environment: %s\n", err)
				// 	return
				// }
				// fmt.Printf("Virtual environment activated for test: %s\n", testConfig.Name)

				// Update the command to run the test using the virtual environment
				testConfig.Executable = "/tmp/" + testConfig.Name + "/" + testConfig.WorkingDir + "/venv/bin/" + testConfig.Executable

			}

			// Execute the test command
			// fmt.Printf("Executing test: %s\n", testConfig.Name)
			// // Here you would execute the test command, e.g., using os/exec
			// runString := fmt.Sprintf("cd /tmp/%s/%s && %s", testConfig.Name, testConfig.WorkingDir, testConfig.Command)
			runString := fmt.Sprintf("cd /tmp/%s/%s && %s", testConfig.Name, testConfig.WorkingDir, testConfig.Executable+" "+testConfig.Command)
			cmd := exec.Command("bash", "-c", runString)

			// Print the output of the command
			output, err := cmd.Output()
			if err != nil {
				fmt.Printf("Error executing test command: %s\n", err)
				return
			}
			fmt.Printf("Output of test command: %s\n", output)
			fmt.Printf("Test %s executed successfully\n", testConfig.Name)

		})
		fmt.Printf("Scheduled test: %s every %d seconds\n", testConfig.Name, testConfig.Frequency)
	}

	// s.Every(duration).Do(func() {
	// 	// Implement logic to pull updates and update config
	// 	fmt.Println("Checking for config updates...")
	// 	// ...
	// })

	s.StartBlocking()
}

func cloneOrUpdateRepo(repoURL string, user string, token string, dirName string) error {
	// Check if the directory exists
	if _, err := os.Stat("/tmp/" + dirName); err == nil {
		// Open the existing repository
		repo, err := git.PlainOpen("/tmp/" + dirName)
		if err != nil {
			fmt.Println("Error opening repository:", err)
			return err
		}

		// Pull the latest changes
		w, err := repo.Worktree()
		if err != nil {
			fmt.Println("Error getting worktree:", err)
			return err
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
				return err
			} else {
				fmt.Println("Repository is already up-to-date")
			}
		} else {
			fmt.Println("Successfully pulled latest changes")
		}
	} else if os.IsNotExist(err) {
		// If the directory does not exist, clone the repository
		repo, err := git.PlainClone("/tmp/"+dirName, false, &git.CloneOptions{
			URL:      repoURL,
			Progress: os.Stdout,
			Auth: &http.BasicAuth{
				Username: user,
				Password: token,
			},
		})
		if err != nil {
			fmt.Println("Error cloning repository:", err)
			return err
		}
		fmt.Println(repo)
	}
	return nil
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
	err := cloneOrUpdateRepo(configRepo, gitConfigUser, gitConfigToken, "runner-config")
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
	fmt.Println(config)
	return config, nil
}

func readAndParseConfig(filePath string) (Config, error) {
	// read the file, unmarshal the yaml, and return the result.

	cfg, err := yamlcfg.Parse[Config](filePath)
	if err != nil {
		return Config{}, err
	}

	// print the parsed config for debugging
	fmt.Printf("Parsed config: %+v\n", cfg)

	return *cfg, nil

}
