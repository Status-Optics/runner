# Status-Optics runner
This is the runner for the Status-Optics project. The runner is responsible for pulling the test code from git, running the tests, and reporting the results back to the defined endpoint(s).

## Runner Configuration
```yaml
# Runner Configuration (runner.yaml)

runner:
  name: "my-app-monitor-runner" # Optional: A descriptive name for the runner
  version: "1.0.0" # Optional: Runner version

git:
  repository: "https://github.com/your-org/your-test-repo.git" # Required: Git repository URL
  branch: "main" # Optional: Git branch (default: main)
  pull_interval: "60s" # Required for cron-based polling (e.g., "1m", "30s", "1h")
  webhook_enabled: false # Optional: Enable webhook support (default: false)
  webhook_secret: "your-secret-webhook-token" # Required if webhook_enabled: true

test:
  language: "python" # Required: Language of the test scripts (e.g., "python", "javascript", "bash")
  command: "pytest tests/" # Required: Command to execute the tests (e.g., "pytest", "npm test", "sh run_tests.sh")
  working_directory: "/app/tests" # Optional: Working directory for test execution

reporting:
  endpoints: # Required: List of reporting endpoints
    - type: "datadog"
      api_key: "your-datadog-api-key"
      app_key: "your-datadog-app-key"
      metric_prefix: "app.monitor."
    - type: "s3"
      bucket: "your-s3-bucket"
      region: "us-east-1"
      access_key_id: "your-access-key-id"
      secret_access_key: "your-secret-access-key"
      path: "results/"
    - type: "saas" # if using the SaaS service.
      api_url: "https://your-saas-service.com/api/results"
      api_token: "your-saas-api-token"
  format: "json" # Optional: Format of the test results (e.g., "json", "xml", "text")

saas: # Configuration for SaaS integration (optional)
  enabled: false # Optional: Enable SaaS integration (default: false)
  api_url: "https://your-saas-service.com/api/runner" # Required if saas.enabled: true
  runner_id: "unique-runner-id" # Required if saas.enabled: true
  api_token: "your-saas-api-token" # Required if saas.enabled: true

logging:
  level: "INFO" # Optional: Logging level (e.g., "DEBUG", "INFO", "WARNING", "ERROR")
  format: "text" # Optional: Logging format (e.g., "json", "text")
```

