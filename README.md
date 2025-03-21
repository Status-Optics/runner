# Status-Optics runner
This is the runner for the Status-Optics project. The runner is responsible for pulling the test code from git, running the tests, and reporting the results back to the defined endpoint(s).

## Environment Variables needed to bootstrap the runner
RUNNER_CONFIG_REPO - The URL of the git repository containing the runner configuration.
RUNNER_CONFIG_BRANCH - The branch of the git repository containing the runner configuration.
RUNNER_CONFIG_FILE - The path to the runner configuration file in the git repository.


## Runner Configuration
```yaml
# Runner Configuration (runner.yaml)

runner:
  name: "my-app-monitor-runner"
  version: "1.0.0"
  log_level: "error"

tests:
  - name: "api-health-check"
    language: "python"
    setup_cmd: "pip3 install -r requirements.txt"
    executable: "python3"
    command: "python-httpbin-get.py"
    frequency: 60
    source:
      repository: "https://github.com/Status-Optics/examples.git"
      branch: "main"
      directory: "python"
      get_user: "${GIT_USER_ENV_VAR_NAME}"
      get_token: "${GIT_TOKEN_ENV_VAR_NAME}"

reporting:
  endpoints:
    - type: "stdout"
      format: "json"




```

