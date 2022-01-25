package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var build = "1" // build number set at compile time
func main() {
	app := cli.NewApp()
	app.Name = "Drone-Sonar-Plugin"
	app.Usage = "Drone plugin to integrate with SonarQube and check for Quality Gate."
	app.Action = run
	app.Version = fmt.Sprintf("1.0.%s", build)
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "key",
			Usage:  "project key",
			EnvVar: "PLUGIN_SONAR_KEY",
		},
		cli.StringFlag{
			Name:   "name",
			Usage:  "project name",
			EnvVar: "PLUGIN_SONAR_NAME",
		},
		cli.StringFlag{
			Name:   "host",
			Usage:  "SonarQube host",
			EnvVar: "PLUGIN_SONAR_HOST",
		},
		cli.StringFlag{
			Name:   "token",
			Usage:  "SonarQube token",
			EnvVar: "PLUGIN_SONAR_TOKEN",
		},
		cli.StringFlag{
			Name:   "ver",
			Usage:  "Project version",
			EnvVar: "PLUGIN_BUILD_NUMBER",
		},
		cli.StringFlag{
			Name:   "branch",
			Usage:  "Project branch",
			EnvVar: "PLUGIN_BRANCH",
		},
		cli.StringFlag{
			Name:   "timeout",
			Usage:  "Web request timeout",
			Value:  "300",
			EnvVar: "PLUGIN_TIMEOUT",
		},
		cli.StringFlag{
			Name:   "sources",
			Usage:  "analysis sources",
			Value:  ".",
			EnvVar: "PLUGIN_SOURCES",
		},
		cli.StringFlag{
			Name:   "inclusions",
			Usage:  "code inclusions",
			EnvVar: "PLUGIN_INCLUSIONS",
		},
		cli.StringFlag{
			Name:   "exclusions",
			Usage:  "code exclusions",
			EnvVar: "PLUGIN_EXCLUSIONS",
		},
		cli.StringFlag{
			Name:   "level",
			Usage:  "log level",
			Value:  "INFO",
			EnvVar: "PLUGIN_LEVEL",
		},
		cli.StringFlag{
			Name:   "showProfiling",
			Usage:  "showProfiling during analysis",
			Value:  "false",
			EnvVar: "PLUGIN_SHOWPROFILING",
		},
		cli.BoolFlag{
			Name:   "branchAnalysis",
			Usage:  "execute branchAnalysis",
			EnvVar: "PLUGIN_BRANCHANALYSIS",
		},
		cli.BoolFlag{
			Name:   "usingProperties",
			Usage:  "using sonar-project.properties",
			EnvVar: "PLUGIN_USINGPROPERTIES",
		},
		cli.StringFlag{
			Name:   "binaries",
			Usage:  "Java Binaries",
			EnvVar: "PLUGIN_BINARIES,JAVA_BINARIES",
		},
		cli.StringFlag{
			Name:   "quality",
			Usage:  "Quality Gate",
			EnvVar: "SONAR_QUALITYGATE,PLUGIN_QUALITYGATE",
			Value:  "OK",
		},
		cli.StringFlag{
			Name:   "quality_gate_enabled",
			Usage:  "true or false - stop pipeline if sonar quality gate conditions are not met",
			Value:  "true",
			EnvVar: "PLUGIN_SONAR_QUALITY_ENABLED",
		},
		cli.StringFlag{
			Name:   "qualitygate_timeout",
			Usage:  "number in seconds for timeout",
			Value:  "300",
			EnvVar: "PLUGIN_SONAR_QUALITYGATE_TIMEOUT",
		},
		cli.StringFlag{
			Name:   "artifact_file",
			Usage:  "Artifact file location that will be generated by the plugin. This file will include information of docker images that are uploaded by the plugin.",
			Value:  "artifact.json",
			EnvVar: "PLUGIN_ARTIFACT_FILE",
		},
		cli.StringFlag{
			Name:   "javascript_icov_reportPath",
			Usage:  "Sonar Javascript Icov Report Path parameter",
			Value:  "",
			EnvVar: "PLUGIN_JAVASCRIPT_ICOV_REPORTPATH",
		},
		cli.StringFlag{
			Name:   "java_coveragePlugin",
			Usage:  "Sonar Java Plugin parameter",
			Value:  "",
			EnvVar: "PLUGIN_JAVASCRIPT_ICOV_REPORTPATH",
		},
		cli.StringFlag{
			Name:   "jacoco_reportPath",
			Usage:  "Sonar Javascript Jacoco Report Path parameter",
			Value:  "",
			EnvVar: "PLUGIN_JAVASCRIPT_ICOV_REPORTPATH",
		},
	}
	app.Run(os.Args)
}
func run(c *cli.Context) {
	plugin := Plugin{
		Config: Config{
			Key:                  c.String("key"),
			Name:                 c.String("name"),
			Host:                 c.String("host"),
			Token:                c.String("token"),
			Version:              c.String("ver"),
			Branch:               c.String("branch"),
			Timeout:              c.String("timeout"),
			Sources:              c.String("sources"),
			Inclusions:           c.String("inclusions"),
			Exclusions:           c.String("exclusions"),
			Level:                c.String("level"),
			ShowProfiling:        c.String("showProfiling"),
			BranchAnalysis:       c.Bool("branchAnalysis"),
			UsingProperties:      c.Bool("usingProperties"),
			Binaries:             c.String("binaries"),
			Quality:              c.String("quality"),
			QualityEnabled:       c.String("quality_gate_enabled"),
			ArtifactFile:         c.String("artifact_file"),
			QualityTimeout:       c.String("qualitygate_timeout"),
			JavascitptIcovReport: c.String("javascript_icov_reportPath"),
			JavaCoveragePlugin:   c.String("java_coveragePlugin"),
			JacocoReportPath:     c.String("jacoco_reportPath"),
		},
	}
	os.Setenv("TOKEN", base64.StdEncoding.EncodeToString([]byte(c.String("token")+":")))
	if err := plugin.Exec(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
