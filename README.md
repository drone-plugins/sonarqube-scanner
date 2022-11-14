# Harness Drone/CIE SonarQube Plugin with Quality Gate

The plugin of Harness Drone/CIE to integrate with SonarQube (previously called Sonar), which is an open source code quality management platform and check the report results for status OK.

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/SonarResult.png" alt="Results" width="800"/>


Detail Informations/tutorials Parameteres: [DOCS.md](DOCS.md).


### Build process
build go binary file: 
`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o sonarqube-scanner`

build docker image
`docker build -t plugins/sonarqube-scanner .`


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

### Pipeline example
```yaml
- step:
    type: Plugin
    name: "Check Sonar "
    identifier: Check_Sonar
    spec:
        connectorRef: account.DockerHubDiego
        image: plugins/sonarqube-scanner:linux-amd64
        reports:
            type: JUnit
            spec:
                paths:
                    - "**/**/*.xml"
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
    image: plugins/sonarqube-scanner:linux-amd64
    reports:
        type: JUnit
        spec:
            paths:
                - "**/**/*.xml"
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

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/Sonar-CIE.png" alt="Plugin Configuration" width="400"/>

<img src="https://github.com/drone-plugins/sonarqube-scanner/blob/main/SonarResultConsole.png" alt="Console Results" width="800"/>
