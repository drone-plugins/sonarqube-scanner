[![Quality Gate Status](https://sonar.harness-demo.site/api/project_badges/measure?project=sonarqube-scanner&metric=alert_status&token=sqb_44744288d1f30ee517e04dbf69e29f836cc8f45b)](https://sonar.harness-demo.site/dashboard?id=sonarqube-scanner)

# Harness Drone/CIE SonarQube Plugin with Quality Gate

This plugin is designed to run SonarQube scans and handle the results and convert it to JUnit Format. It's written in Go and check the report results for status OK.

## Main Features - v2.2.1

- Execute SonarQube scans and handle the results
- Generate JUnit reports based on the scan results
- Quality Gate status reporting + Metrics
- Skip Scan and only check for quality Gate Status of a specific analysisId or last analysis

Obs: USe branch and pr_key params for accuracy results matches when skiping the scan

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/sonar-result-v2.png" alt="Results" width="800"/>

### Simple Pipeline example

```yaml
- step:
    type: Plugin
    name: "Check Sonar"
    identifier: run_sonar
    spec:
        connectorRef: account.DockerHubDiego
        image: plugins/sonarqube-scanner:v2.2.1
        reports:
            type: JUnit
            spec:
                paths:
                  - "SonarResults.xml"
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
    image: plugins/sonarqube-scanner:v2.2.1
    reports:
        type: JUnit
        spec:
            paths:
              - "SonarResults.xml"
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
        image: plugins/sonarqube-scanner:v2.2.1
        reports:
            type: JUnit
            spec:
                paths:
                  - "SonarResults.xml"
        privileged: false
        settings:
            sonar_host: https://sonarcloud.io
            sonar_token: 66778345yourToken817f0deee3daa7868c431433
            sonar_name: sonar-project-name
            sonar_key: sonar-project-key
            skip_scan: true
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
- `ver`: The version of the project.
  - Example: `"ver": "1.0.0"`
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
  - Example: `"coverage_exclusion": "*.test.go"`
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

Detail Informations/tutorials Parameteres: [DOCS.md](DOCS.md).

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
