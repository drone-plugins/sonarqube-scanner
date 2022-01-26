---
date: 2021-09-26T13:36:00+00:00
title: SonarQube
author: diegopereiraeng
tags: [ Sonar, SonarQube, Analysis, report ]
logo: sonarqube.svg
repo: drone-plugins/sonarqube-scanner
image: drone-plugins/sonarqube-scanner:latest
---

This plugin can scan your code quality and post the analysis report to your SonarQube server. SonarQube (previously called Sonar), is an open source code quality management platform.

The below pipeline configuration demonstrates simple usage:

```yaml
steps:
- name: code-analysis
  image: drone-plugins/sonarqube-scanner:latest
  settings:
    sonar_host:
      from_secret: sonar_host
    sonar_token:
      from_secret: sonar_token
```

Customized parameters could be specified:

```diff
  steps:
  - name: code-analysis
    image: drone-plugins/sonarqube-scanner
    settings:
      sonar_host:
        from_secret: sonar_host
      sonar_token:
        from_secret: sonar_token
      sonar_name: project-harness
      sonar_key: project-harness
+     ver: 1.0
+     timeout: 20
+     sources: .
+     level: DEBUG
+     showProfiling: true
+     exclusions: **/static/**/*,**/dist/**/*.js
+     usingProperties: false
+     binaries: .
```

# Secret Reference

Safety first, the host and token are stored in Drone Secrets.
* `sonar_host`: Host of SonarQube with schema(http/https).
* `sonar_token`: User token used to post the analysis report to SonarQube Server. Click User -- My Account -- Security -- Generate Tokens.


# Parameter Reference

* `sonar_name`: Sonar Project NAme.
* `sonar_key`: Sonar Project Key.
* `sonar_qualitygate_timeout`: Timeout in seconds for Sonar Scan.
* `artifact_file`: Timeout in seconds for Sonar Scan.
* `sonar_quality_enabled`: True to block pipeline if sonar quality gate conditions are not met.
* `branch`: Branch for analysis.
* `build_number`: Build Version.

* `build_version`: Code version, Default value `DRONE_BUILD_NUMBER`.
* `timeout`: Default seconds `60`.
* `sources`: Comma-separated paths to directories containing source files. 
* `inclusions`: Comma-delimited list of file path patterns to be included in analysis. When set, only files matching the paths set here will be included in analysis.
* `exclusions`: Comma-delimited list of file path patterns to be excluded from analysis. Example: `**/static/**/*,**/dist/**/*.js`.
* `level`: Control the quantity / level of logs produced during an analysis. Default value `INFO`. 
    * DEBUG: Display INFO logs + more details at DEBUG level.
    * TRACE: Display DEBUG logs + the timings of all ElasticSearch queries and Web API calls executed by the SonarQube Scanner.
* `showProfiling`: Display logs to see where the analyzer spends time. Default value `false`
* `branchAnalysis`: Pass currently analysed branch to SonarQube. (Must not be active for initial scan!) Default value `false`


* `usingProperties`: Using the `sonar-project.properties` file in root directory as sonar parameters. (Not include `sonar_host` and
`sonar_token`.) Default value `false`






# Javascript Parameters

* `javascript_icov_reportPath`: Path to coverage report (-Dsonar.javascript.lcov.reportPath)

# Notes

* projectKey: `PLUGIN_SONAR_KEY`
* projectName: `PLUGIN_SONAR_NAME`
* You could also add a file named `sonar-project.properties` at the root of your project to specify parameters.

Code repository: [drone-plugins/sonarqube-scanner](https://github.com/drone-plugins/sonarqube-scanner).  
SonarQube Parameters: [Analysis Parameters](https://docs.sonarqube.org/display/SONAR/Analysis+Parameters)

# Test your SonarQube Server:

Replace the parameter values with your own：

```commandline
sonar-scanner \
  -Dsonar.projectKey=Harness:cie \
  -Dsonar.sources=. \
  -Dsonar.projectName=Harness/cie \
  -Dsonar.projectVersion=1.0 \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.login=60878847cea1a31d817f0deee3daa7868c431433
```
