# Status-Optics runner
This is the runner for the Status-Optics project. The runner is responsible for pulling the test code from git, running the tests, and reporting the results back to the defined endpoint(s).

## Environment Variables needed to bootstrap the runner
RUNNER_CONFIG_REPO - The URL of the git repository containing the runner configuration.
RUNNER_CONFIG_BRANCH - The branch of the git repository containing the runner configuration.
RUNNER_CONFIG_PATH - The path to the runner configuration file in the git repository.


## Runner Configuration
```yaml
# Runner Configuration (runner.yaml)

runner:
  name: "my-app-monitor-runner"
  version: "1.0.0"

tests:
  - name: "api-health-check"
    language: "python"
    command: "pytest tests/api_health_test.py"
    working_directory: "/app/tests"
    source:
      repository: "https://github.com/your-org/your-test-repo.git"
      branch: "main"
      pull_interval: "60s"
      webhook_enabled: false
      webhook_secret: $["git-token-env-var-name"]
  - name: "performance-test"
    language: "javascript"
    command: "npm run performance"
    working_directory: "/app/performance"
    source:
      repository: "https://github.com/your-org/your-test-repo.git"
      branch: "main"
      pull_interval: "60s"
      webhook_enabled: false
      webhook_secret: $["git-token-env-var-name"]
  - name: "database-test"
    language: "bash"
    command: "sh run_db_tests.sh"
    working_directory: "/app/db"
    source:
      repository: "https://github.com/your-org/your-test-repo.git"
      branch: "main"
      pull_interval: "60s"
      webhook_enabled: false
      webhook_secret: $["git-token-env-var-name"]

reporting:
  endpoints:
    - type: "datadog"
      api_key: $["your-datadog-api-key"]
      app_key: $["your-datadog-app-key"]
      metric_prefix: "app.monitor."
    - type: "s3"
      bucket: "your-s3-bucket"
      region: "us-east-1"
      access_key_id: $["your-access-key-id"]
      secret_access_key: $["your-secret-access-key"]
      path: "results/"
    - type: "director"
      api_url: "https://your-director-service.com/api/results"
      api_token: "your-director-api-token"
  format: "json"

director:
  enabled: false
  api_url: "https://your-director-service.com/api/runner"
  runner_id: $["unique-runner-id"]
  api_token: $["your-director-api-token"]

logging:
  level: "INFO"
  format: "text"
```

