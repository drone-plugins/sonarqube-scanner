# Harness Drone/CIE SonarQube Plugin with Quality Gate

This plugin is designed to run SonarQube scans, handle the results, and convert them to JUnit format. It's written in Go and checks the report results for status OK.

## Main Features - v2.4.2

- **New Parameter: `sonar_config_file`**
  - **Type**: Boolean
  - **Description**: If set to true, the plugin will utilize the specified `sonar-project.properties` file for the SonarQube analysis, if it exists. sonar_token param still mandatory.
  - **Environment Variable**: `PLUGIN_SONAR_CONFIG_FILE`
  - **Allowed Values**: `"true"`, `"false"`

- **New Parameter: `sonar_config_file_override`**
  - **Type**: Boolean
  - **Description**: If set to true, the plugin will use the `sonar-project.properties` file and allow overriding of project key setting.
  - **Environment Variable**: `PLUGIN_SONAR_CONFIG_FILE_OVERRIDE`
  - **Allowed Values**: `"true"`, `"false"`

- **New Parameter: `quality_gate_error_exit_code`**
  - **Type**: Integer
  - **Description**: Specifies the "exit code" error when the quality gate fails. Default is `5`.
  - **Environment Variable**: `PLUGIN_QUALITY_GATE_ERROR_EXIT_CODE`
  - **Default Value**: `5`

### Example

```yaml
- step:
    type: Plugin
    name: "Sonar Scan"
    identifier: run_sonar
    spec:
        connectorRef: account.DockerHubDiego
        image: plugins/sonarqube-scanner:v2.4.2
        reports:
            type: JUnit
            spec:
                paths:
                  - "sonarResults.xml"
        privileged: false
        settings:
            sonar_host: https://mysonar.com
            sonar_token: <+secrets.getValue("sonar_diego")>
            sonar_config_file: "true"
```

## Main Features - v2.4.1

- Sonar CLI downgraded from 6.0.0.4432 to 5.0.1.3006
- Execute SonarQube scans and handle the results
- Generate JUnit reports based on the scan results
- Quality Gate status reporting + Metrics
- Skip Scan and only check for Quality Gate status of a specific `analysisId` or last analysis
- Waiting for Analysis and QualityGate now skips the wait if set to false - thanks @kangguru
- Added `SONAR_SCANNER_OPTS` as a parameter, transforming into this env var during execution for Sonar JVM params - check the parameter detail section below

**Note**: Use `branch` and `pr_key` parameters for accurate results matching when skipping the scan.

![Results](https://github.com/drone-plugins/sonarqube-scanner/blob/main/sonar-result-v2.png)


### Simple Pipeline example

```yaml
- step:
    type: Plugin
    name: "Check Sonar"
    identifier: run_sonar
    spec:
        connectorRef: account.DockerHubDiego
        image: plugins/sonarqube-scanner:v2.4.2
        reports:
            type: JUnit
            spec:
                paths:
                  - "sonarResults.xml"
        privileged: false
        settings:
            sonar_host: http://34.100.11.50
            sonar_token: 60878847cea1a31d817f0deee3daa7868c431433
            sources: "."
            binaries: "."
            sonar_name: sonarqube-scanner
            sonar_key: sonarqube-scanner
```

### Full config step Example - (thanks @Ryan Nelson)

```yaml
type: Plugin
spec:
    connectorRef: <+input>
    image: plugins/sonarqube-scanner:v2.4.2
    reports:
        type: JUnit
        spec:
            paths:
              - "sonarResults.xml"
    settings:
        sonar_key: <+input>
        sonar_name: <+input>
        sonar_host: <+input>
        sonar_token: <+input>
        build_number: <+input>
        branch: <+codebase.branch>
        timeout: <+input>
        sources: .
        inclusions: <+input>
        exclusions: <+input>
        level: <+input>
        showprofiling: <+input>.allowedValues("true","false"
        branchanalysis: <+input>.allowedValues("true","false")
        usingproperties: <+input>.allowedValues("true","false")
        binaries: <+input>
        sonar_qualitygate: OK
        sonar_quality_enabled: <+input>.allowedValues("true","false")
        sonar_qualitygate_timeout: <+input>
        artifact_file: <+input>
        javascript_icov_reportpath: <+input>
        java_coverage_plugin: <+input>
        jacoco_report_path: <+input>
        
```

### Skip Scan - Pipeline example
```yaml
- step:
    type: Plugin
    name: "Check Sonar Quality Gate"
    identifier: check_sonar
    spec:
        connectorRef: account.DockerHubDiego
        image: plugins/sonarqube-scanner:v2.4.2
        reports:
            type: JUnit
            spec:
                paths:
                  - "sonarResults.xml"
        privileged: false
        settings:
            sonar_host: https://sonarcloud.io
            sonar_token: 66778345yourToken817f0deee3daa7868c431433
            sonar_name: sonar-project-name
            sonar_key: sonar-project-key
            skip_scan: true
```

### DRONE Pipeline example
```yaml
kind: pipeline
name: default

steps:
- name: perform-code-analysis
  image: plugins/sonarqube-scanner:v2.4.2
  settings:
    sonar_host: http://34.100.11.50
    sonar_token:
      from_secret: sonar_token
    sources: .
    binaries: .
    sonar_name: sonarqube-scanner
    sonar_key: sonarqube-scanner
```



### Configuration Parameters

- `key`: The project key in SonarQube.
  - Example: `"key": "your-project-key"`
- `name`: The project name in SonarQube.
  - Example: `"name": "your-project-name"`
- `host`: The URL of the SonarQube server.
  - Example: `"host": "https://sonarqube.example.com"`
- `token`: The token for authenticating with the SonarQube server.
  - Example: `"token": "your-sonarqube-token"`
- `build_number`: The version of the project.
  - Example: `"build_number": "1.0.0"`
- `workspace`: The workspace folder that will be passed to SonarQube CLI in case you are not cloning or scaning the /harness folder directly.
  - Example: `"workspace": "/harness/subfolder"`
- `branch`: The branch of the project. This parameter is used to specify the branch of your codebase that the results should be matched with. If you're working on multiple branches, it's important to specify the correct branch to ensure that you're looking at the correct set of results.
  - Example: `"branch": "master"`
- `timeout`: The timeout for the Sonar scanner.
  - Example: `"timeout": "300"`
- `sources`: The paths for the source directories, separated by commas.
  - Example: `"sources": "src"`
- `inclusions`: The files to be included in the analysis.
  - Example: `"inclusions": "*.go, *.java"`
- `exclusions`: The files to be excluded from the analysis.
  - Example: `"exclusions": "**/test/**/*.*,**/*.test.go"`
- `level`: The logging level.
  - Example: `"level": "INFO"`
- `showProfiling`: Enable profiling during analysis.
  - Example: `"showProfiling": "true"`
- `branchAnalysis`: Execute branch analysis.
  - Example: `"branchAnalysis": "true"`
- `usingProperties`: Use `sonar-project.properties`.
  - Example: `"usingProperties": "true"`
- `binaries`: Java binaries.
  - Example: `"binaries": "/path/to/binaries"`
- `quality`: Quality Gate.
  - Example: `"quality": "OK"`
- `quality_gate_enabled`: Stop pipeline if Sonar quality gate conditions are not met.
  - Example: `"quality_gate_enabled": "true"`
- `qualitygate_timeout`: Number in seconds for timeout.
  - Example: `"qualitygate_timeout": "300"`
- `artifact_file`: Artifact file location that will be generated by the plugin. This file will include information of Docker images that are uploaded by the plugin.
  - Example: `"artifact_file": "artifact.json"`
- `output-file`: Output file location that will be generated by the plugin. This file will include information that is exported by the plugin.
  - Example: `"output-file": "/path/to/output/file"`
- `javascript_icov_reportPath`: Sonar JavaScript Icov Report Path parameter.
  - Example: `"javascript_icov_reportPath": "/path/to/icov/report"`
- `java_coverage_plugin`: Sonar Java Plugin parameter.
  - Example: `"java_coverage_plugin": "jacoco"`
- `jacoco_report_path`: Sonar Jacoco Report Path parameter.
  - Example: `"jacoco_report_path": "/path/to/jacoco/report"`
- `ssl_keystore_pwd`: Java Keystore Password.
  - Example: `"ssl_keystore_pwd": "your-keystore-password"`
- `cacerts_location`: Java Truststore Location (cacerts).
  - Example: `"cacerts_location": "/path/to/cacerts"`
- `junit_reportpaths`: JUnit Report Paths.
  - Example: `"junit_reportpaths": "/path/to/junit/reports"`
- `source_encoding`: Source Encoding.
  - Example: `"source_encoding": "UTF-8"`
- `tests`: Sonar Tests.
  - Example: `"tests": "/path/to/tests"`
- `java_test`: Java Test.
  - Example: `"java_test": "/path/to/java/test"`
- `pr_key`: Pull Request Key.
  - Example: `"pr_key": "123"`
- `pr_branch`: PR Branch.
  - Example: `"pr_branch": "your-pr-branch"`
- `pr_base`: PR Base.
  - Example: `"pr_base": "your-pr-base"`
- `coverage_exclusion`: Sonar coverage exclusions.
  - Example: `"coverage_exclusion": "**/test/**/*.*,**/*.test.go"`
- `java_source`: Sonar Java source.
  - Example: `"java_source": "1.8"`
- `java_libraries`: Sonar Java libraries.
  - Example: `"java_libraries": "/path/to/libraries"`
- `surefire_reportsPath`: Sonar surefire reportsPath.
  - Example: `"surefire_reportsPath": "/path/to/surefire/reports"`
- `typescript_lcov_reportPaths`: Sonar TypeScript lcov reportPaths.
  - Example: `"typescript_lcov_reportPaths": "/path/to/typescript/lcov/reports"`
- `verbose`: Sonar verbose.
  - Example: `"verbose": "true"`
- `custom_jvm_params`: JVM parameters. Use comma for multiple parameters.
  - Example: `"custom_jvm_params": "-Dsonar.java.source='value_you_want'"`
- `taskid`: Sonar analysis taskId.
  - Example: `"taskid": "your-task-id"`
- `skip_scan`: Skip Sonar analysis scan - get last analysis automatically.
  - Example: `"skip_scan": true`
- `SONAR_SCANNER_OPTS`: pass any Sonar JVM param as env var during execution.
  - Example: `"SONAR_SCANNER_OPTS": "--add-opens java.base/sun.nio.ch=ALL-UNNAMED --add-opens java.base/java.io=ALL-UNNAMED"`

- **`sonar_config_file`**:
  - **Type**: Boolean
  - **Description**: Use `sonar-project.properties` if available.
  - **Environment Variable**: `PLUGIN_SONAR_CONFIG_FILE`
  - **Allowed Values**: `"true"`, `"false"`

- **`sonar_config_file_override`**:
  - **Type**: Boolean
  - **Description**: Use `sonar-project.properties` if available and allow overriding of host, login, and/or project key settings.
  - **Environment Variable**: `PLUGIN_SONAR_CONFIG_FILE_OVERRIDE`
  - **Allowed Values**: `"true"`, `"false"`

- **`quality_gate_error_exit_code`**:
  - **Type**: Integer
  - **Description**: Specifies the "exit code" error when the quality gate fails. Default is `5`.
  - **Environment Variable**: `PLUGIN_QUALITY_GATE_ERROR_EXIT_CODE`
  - **Default Value**: `5` 

Detail Informations/tutorials Parameteres: [DOCS.md](DOCS.md).

### Sonar Token

Use Global analysis tokens

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/global-analysis-token.png" alt="Plugin Configuration" width="400"/>

### Build Process

build go binary file: 
```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o sonarqube-scanner
```

build docker image
```
docker build -t plugins/sonarqube-scanner .
```


### Testing the docker image:
```commandline
docker run --rm \
  -e DRONE_REPO=test \
  -e PLUGIN_SOURCES=. \
  -e SONAR_HOST=http://localhost:9000 \
  -e SONAR_TOKEN=60878847cea1a31d817f0deee3daa7868c431433 \
  -e PLUGIN_SONAR_KEY=project-sonar \
  -e PLUGIN_SONAR_NAME=project-sonar \
  plugins/sonarqube-scanner
```

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/Sonar-CIE.png" alt="Plugin Configuration" width="400"/>

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/SonarResultConsole.png" alt="Console Results" width="800"/>
