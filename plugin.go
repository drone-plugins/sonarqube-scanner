package main

// Standard library imports
import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml"
	"github.com/sirupsen/logrus"
)

// External library imports

// Global variables
var (
	// netClient is used for making HTTP requests.
	netClient *http.Client

	// projectKey represents the key of the project.
	projectKey = ""

	// sonarDashStatic is a static string used in the dashboard URL.
	sonarDashStatic = "/dashboard?id="
	//https://sonar.dfinsolutions.com/dashboard?id=dfinsolutions_Saturn-UI_AYezvlRKNrcjU-xpGTBl&pullRequest=1244
	// basicAuth is the basic authentication string.
	basicAuth = "Basic "
)

const (
	lineBreak  = "----------------------------------------------"
	lineBreak2 = "|----------------------------------------------------------------|"
)

type (
	Config struct {
		Key                        string
		Name                       string
		Host                       string
		Token                      string
		Version                    string
		Branch                     string
		Sources                    string
		Timeout                    string
		Inclusions                 string
		Exclusions                 string
		Level                      string
		ShowProfiling              string
		BranchAnalysis             bool
		UsingProperties            bool
		Binaries                   string
		Quality                    string
		QualityEnabled             string
		QualityTimeout             string
		ArtifactFile               string
		JavascitptIcovReport       string
		JavaCoveragePlugin         string
		JacocoReportPath           string
		SSLKeyStorePassword        string
		CacertsLocation            string
		JunitReportPaths           string
		SourceEncoding             string
		SonarTests                 string
		JavaTest                   string
		PRKey                      string
		PRBranch                   string
		PRBase                     string
		CoverageExclusion          string
		JavaSource                 string
		JavaLibraries              string
		SurefireReportsPath        string
		TypescriptLcovReportPaths  string
		Verbose                    string
		CustomJvmParams            string
		TaskId                     string
		SkipScan                   bool
		WaitQualityGate            bool
		Workspace                  string
		SonarOPS                   string
		UseSonarConfigFile         bool
		UseSonarConfigFileOverride bool
		QualityGateErrorExitCode   int
	}
	Output struct {
		OutputFile string // File where plugin output are saved
	}
	// SonarReport it is the representation of .scannerwork/report-task.txt //
	SonarReport struct {
		ProjectKey   string `toml:"projectKey"`
		ServerURL    string `toml:"serverUrl"`
		DashboardURL string `toml:"dashboardUrl"`
		CeTaskID     string `toml:"ceTaskId"`
		CeTaskURL    string `toml:"ceTaskUrl"`
	}
	Plugin struct {
		Config Config
		Output Output // Output file content
	}
	// TaskResponse Give Compute Engine task details such as type, status, duration and associated component.
	TaskResponse struct {
		Task struct {
			ID                 string   `json:"id"`
			Type               string   `json:"type"`
			ComponentID        string   `json:"componentId"`
			ComponentKey       string   `json:"componentKey"`
			ComponentName      string   `json:"componentName"`
			ComponentQualifier string   `json:"componentQualifier"`
			AnalysisID         string   `json:"analysisId"`
			Status             string   `json:"status"`
			SubmittedAt        string   `json:"submittedAt"`
			SubmitterLogin     string   `json:"submitterLogin"`
			StartedAt          string   `json:"startedAt"`
			ExecutedAt         string   `json:"executedAt"`
			ExecutionTimeMs    int      `json:"executionTimeMs"`
			HasScannerContext  bool     `json:"hasScannerContext"`
			WarningCount       int      `json:"warningCount"`
			Warnings           []string `json:"warnings"`
		} `json:"task"`
	}

	// ProjectStatusResponse Get the quality gate status of a project or a Compute Engine task
	ProjectStatusResponse struct {
		ProjectStatus struct {
			Status string `json:"status"`
		} `json:"projectStatus"`
	}

	Project struct {
		ProjectStatus Status `json:"projectStatus"`
	}

	Status struct {
		Status            string      `json:"status"`
		Conditions        []Condition `json:"conditions"`
		IgnoredConditions bool        `json:"ignoredConditions"`
		// Periods           []Period    `json:"periods,omitempty"` // some responses don't have this, so it's marked as omitempty
		// Period            *Period     `json:"period,omitempty"` // some responses don't have this, so it's marked as omitempty
	}

	Condition struct {
		Status         string `json:"status"`
		MetricKey      string `json:"metricKey"`
		Comparator     string `json:"comparator"`
		PeriodIndex    int    `json:"periodIndex"`
		ErrorThreshold string `json:"errorThreshold"`
		ActualValue    string `json:"actualValue"`
	}

	Testsuites struct {
		XMLName   xml.Name    `xml:"testsuites"`
		Text      string      `xml:",chardata"`
		TestSuite []Testsuite `xml:"testsuite"`
	}
	Testsuite struct {
		Text     string     `xml:",chardata"`
		Package  string     `xml:"package,attr"`
		Time     int        `xml:"time,attr"`
		Tests    int        `xml:"tests,attr"`
		Errors   int        `xml:"errors,attr"`
		Name     string     `xml:"name,attr"`
		TestCase []Testcase `xml:"testcase"`
	}

	Testcase struct {
		Text      string   `xml:",chardata"`
		Time      int      `xml:"time,attr"`      // Actual Value Sonar
		Name      string   `xml:"name,attr"`      // Metric Key
		Classname string   `xml:"classname,attr"` // The metric Rule
		Failure   *Failure `xml:"failure"`        // Sonar Failure - show results
	}
	Failure struct {
		Text    string `xml:",chardata"`
		Message string `xml:"message,attr"`
	}
)

type AnalysisResponse struct {
	Analyses []struct {
		Key string `json:"key"`
	} `json:"analyses"`
}

func init() {
	netClient = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
}

func TryCatch(f func()) func() error {
	return func() (err error) {
		defer func() {
			if panicInfo := recover(); panicInfo != nil {
				err = fmt.Errorf("%v", panicInfo)
				return
			}
		}()
		f() // calling the decorated function
		return err
	}
}

// displaySummary provides a colorful summary of the results in the terminal.
func displaySummary(total, passed, failed int, errors int, newErrors int, projectJSON []byte) {
	// Calculate the success rate
	var successRate float64

	// Get the path for DRONE_OUTPUT
	droneOutputPath := os.Getenv("DRONE_OUTPUT")
	fmt.Print("\nDRONE_OUTPUT var: " + droneOutputPath + "\n")
	if droneOutputPath == "" {
		fmt.Print("\nError: DRONE_OUTPUT environment variable not set.\n")
		fmt.Print("\nError: Probably you are not running in Harness or Drone.\n")
		// return
	}

	if total != 0 {
		successRate = float64(passed) / float64(total) * 100
	} else {
		successRate = 0 // or any other default value
	}

	os.Setenv("SONAR_RESULT_SUCCESS_RATE", fmt.Sprintf("%.2f", successRate)) // Round to two decimal places and set as an environment variable
	os.Setenv("SONAR_RESULT_TOTAL", fmt.Sprintf("%d", total))                // Set the total number of tests as an environment variable
	os.Setenv("SONAR_RESULT_PASSED", fmt.Sprintf("%d", passed))              // Set the number of passed tests as an environment variable
	os.Setenv("SONAR_RESULT_FAILED", fmt.Sprintf("%d", failed))              // Set the number of failed tests as an environment variable

	// Categorize the results
	var category string
	if successRate >= 90 {
		category = "\033[32mExcellent\033[0m" // Green
	} else if successRate >= 70 {
		category = "\033[1;34mGood\033[0m" // Light Blue
	} else {
		category = "\033[1;31mNeeds Improvement\033[0m" // Light Red
	}

	// Prepare your environment variables
	vars := map[string]string{
		"SONAR_RESULT_SUCCESS_RATE": fmt.Sprintf("%.2f", successRate),
		"SONAR_RESULT_TOTAL":        fmt.Sprintf("%d", total),
		"SONAR_RESULT_PASSED":       fmt.Sprintf("%d", passed),
		"SONAR_RESULT_FAILED":       fmt.Sprintf("%d", failed),
		"SONAR_RESULT_ERRORS":       fmt.Sprintf("%d", errors),
		"SONAR_RESULT_NEW_ERRORS":   fmt.Sprintf("%d", newErrors),
		// "SONAR_RESULT_JSON":         fmt.Sprintf("%d", string(projectJSON)),
	}

	// Write to the .env file
	filePath := fmt.Sprintf(droneOutputPath)
	err := writeEnvFile(vars, filePath)
	if err != nil {
		fmt.Println("Error writing to .env file:", err)
		// return
	}

	fmt.Println("Successfully wrote to .env file")
	// defer file.Close()
	fmt.Println("Successfully closed .env file")
	fmt.Print("\n\n")
	// Display the table
	fmt.Println(lineBreak)
	fmt.Printf("|           STATUS           |      COUNT      |\n")
	fmt.Println(lineBreak)
	fmt.Printf("|      (\033[32mPASSED\033[0m)              |      %d         |\n", passed)
	fmt.Println(lineBreak)
	fmt.Printf("|      (\033[31mFAILED\033[0m)              |      %d         |\n", failed)
	fmt.Println(lineBreak)
	fmt.Printf("|      TOTAL                 |      %d         |\n", total)
	fmt.Println(lineBreak)
	fmt.Printf("\n\nCategorization: %s\n", category)
}

func writeEnvFile(vars map[string]string, outputPath string) error {
	// Use godotenv.Write() to write the vars map to the specified file
	err := godotenv.Write(vars, outputPath)
	if err != nil {
		fmt.Println("Error writing to .env file:", err)
		return err
	}
	fmt.Println("Successfully wrote to .env file")

	// Read the file contents
	content, err := ioutil.ReadFile(outputPath)
	if err != nil {
		fmt.Println("Error reading the .env file:", err)
		return err
	}

	// Print the file contents
	fmt.Println("File contents:")
	fmt.Println(string(content))

	return nil
}

func ParseJunit(projectArray Project, projectName string) Testsuites {
	failed := 0
	total := 0
	testCases := []Testcase{}
	errors := 0
	newErrors := 0

	conditionsArray := projectArray.ProjectStatus.Conditions

	for _, condition := range conditionsArray {
		total += 1
		if condition.Status != "OK" {
			failed += 1
			if strings.HasPrefix(condition.MetricKey, "new_") {
				newErrors += 1
			}
			cond := &Testcase{
				Name:      condition.MetricKey,
				Classname: "Violate if " + condition.ActualValue + " is " + condition.Comparator + " " + condition.ErrorThreshold,
				Failure:   &Failure{Message: "Violated: " + condition.ActualValue + " is " + condition.Comparator + " " + condition.ErrorThreshold},
			}
			testCases = append(testCases, *cond)
		} else {
			cond := &Testcase{
				Name:      condition.MetricKey,
				Classname: "Violate if " + condition.ActualValue + " is " + condition.Comparator + " " + condition.ErrorThreshold,
				Time:      0,
			}
			testCases = append(testCases, *cond)
		}
	}

	// Display the summary
	passed := total - failed // Corrected this line

	os.Setenv("SONAR_RESULT_NEW_ERRORS", fmt.Sprintf("%d", newErrors))  // Set the number of new errors as an environment variable
	os.Setenv("SONAR_RESULT_OVERALL_ERRORS", fmt.Sprintf("%d", errors)) // Set the number of errors as an environment variable

	dashboardLink := os.Getenv("PLUGIN_SONAR_HOST") + sonarDashStatic + os.Getenv("PLUGIN_SONAR_KEY")
	if os.Getenv("PLUGIN_PR_KEY") != "" {
		dashboardLink = os.Getenv("PLUGIN_SONAR_HOST") + sonarDashStatic + os.Getenv("PLUGIN_SONAR_KEY") + "&pullRequest=" + os.Getenv("PLUGIN_PR_KEY")
	} else if os.Getenv("PLUGIN_BRANCHANALYSIS") == "true" {
		dashboardLink = os.Getenv("PLUGIN_SONAR_HOST") + sonarDashStatic + os.Getenv("PLUGIN_SONAR_KEY") + "&branch=" + os.Getenv("PLUGIN_BRANCH")
	}
	SonarJunitReport := &Testsuites{
		TestSuite: []Testsuite{
			Testsuite{
				Time: 13, Package: projectName, Errors: errors, Tests: total, Name: dashboardLink, TestCase: testCases,
			},
		},
	}

	out, _ := xml.MarshalIndent(SonarJunitReport, " ", "  ")
	fmt.Println(string(out))
	fmt.Printf("\n")
	out, _ = xml.MarshalIndent(testCases, " ", "  ")
	fmt.Println(string(out))
	fmt.Printf("\n")

	projectJSON, err := json.Marshal(projectArray)
	if err != nil {
		fmt.Println("Error marshalling project to JSON:", err)
		// Handle error or return something meaningful
	}

	displaySummary(total, passed, failed, errors, newErrors, projectJSON)

	return *SonarJunitReport
}

func GetProjectKey(key string) string {
	projectKey = strings.Replace(key, "/", ":", -1)
	return projectKey
}

func logConfigInfo(configType, configValue string) {
	fmt.Printf("==> %s: %s\n", configType, configValue)
}

func PreFlightGetLatestTaskID(config Config) (string, error) {
	var statusID string
	var err error

	if config.PRKey != "" {
		logConfigInfo("PR Key", config.PRKey)
		statusID, err = getStatusV2("pr", config.PRKey, config.Host, config.Key)
	} else if config.Branch != "" {
		logConfigInfo("Branch", config.Branch)
		statusID, err = getStatusV2("branch", config.Branch, config.Host, config.Key)
	} else {
		logConfigInfo("Project Key", config.Key)
		statusID, err = getStatusID(config.TaskId, config.Host, config.Key)
	}

	if err != nil {
		fmt.Printf("\n\n==> Error getting the latest scanID\n\n")
		fmt.Printf("Error: %s", err.Error())
		return "", err
	}

	return statusID, nil
}

func (p Plugin) Exec() error {
	// Check if the sonar-project.properties file exists in the current directory
	sonarConfigFile := "sonar-project.properties"
	_, err := os.Stat(sonarConfigFile)

	args := []string{}

	// Additional conditions for args
	if len(p.Config.Verbose) >= 1 {
		args = append(args, "-X")
	}

	if len(p.Config.Workspace) >= 1 {
		args = append(args, "-Dsonar.projectBaseDir="+p.Config.Workspace)
	}

	if os.IsNotExist(err) && p.Config.UseSonarConfigFile {
		// If the configuration file does not exist, use the default parameters
		fmt.Println("Configuration file not found. Using default parameters.")
		args = []string{
			"-Dsonar.host.url=" + p.Config.Host,
			"-Dsonar.login=" + p.Config.Token,
		}

		// Map of potential configurations
		configurations := map[string]string{
			"-Dsonar.projectKey":                     p.Config.Key,
			"-Dsonar.projectName":                    p.Config.Name,
			"-Dsonar.projectVersion":                 p.Config.Version,
			"-Dsonar.sources":                        p.Config.Sources,
			"-Dsonar.ws.timeout":                     p.Config.Timeout,
			"-Dsonar.inclusions":                     p.Config.Inclusions,
			"-Dsonar.exclusions":                     p.Config.Exclusions,
			"-Dsonar.log.level":                      p.Config.Level,
			"-Dsonar.showProfiling":                  p.Config.ShowProfiling,
			"-Dsonar.java.binaries":                  p.Config.Binaries,
			"-Dsonar.branch.name":                    p.Config.Branch,
			"-Dsonar.qualitygate.wait":               strconv.FormatBool(p.Config.WaitQualityGate),
			"-Dsonar.qualitygate.timeout":            p.Config.QualityTimeout,
			"-Dsonar.javascript.lcov.reportPaths":    p.Config.JavascitptIcovReport,
			"-Dsonar.coverage.jacoco.xmlReportPaths": p.Config.JacocoReportPath,
			"-Dsonar.java.coveragePlugin":            p.Config.JavaCoveragePlugin,
			"-Dsonar.junit.reportPaths":              p.Config.JunitReportPaths,
			"-Dsonar.sourceEncoding":                 p.Config.SourceEncoding,
			"-Dsonar.tests":                          p.Config.SonarTests,
			"-Dsonar.java.test.binaries":             p.Config.JavaTest,
			"-Dsonar.coverage.exclusions":            p.Config.CoverageExclusion,
			"-Dsonar.java.source":                    p.Config.JavaSource,
			"-Dsonar.java.libraries":                 p.Config.JavaLibraries,
			"-Dsonar.surefire.reportsPath":           p.Config.SurefireReportsPath,
			"-Dsonar.typescript.lcov.reportPaths":    p.Config.TypescriptLcovReportPaths,
			"-Dsonar.verbose":                        p.Config.Verbose,
			"-Dsonar.pullrequest.key":                p.Config.PRKey,
			"-Dsonar.pullrequest.branch":             p.Config.PRBranch,
			"-Dsonar.pullrequest.base":               p.Config.PRBase,
			"-Djavax.net.ssl.trustStorePassword":     p.Config.SSLKeyStorePassword,
			"-Djavax.net.ssl.trustStore":             p.Config.CacertsLocation,
		}

		// Add configurations to args
		for config, value := range configurations {
			if len(value) >= 1 {
				args = append(args, config+"="+value)
			}
		}

		if !p.Config.UsingProperties {
			args = append(args, "-Dsonar.scm.provider=git")
		}

		if len(p.Config.CustomJvmParams) >= 1 {
			params := strings.Split(p.Config.CustomJvmParams, ",")
			args = append(args, params...)
		}

		if len(p.Config.SonarOPS) >= 1 {
			existingOpts := os.Getenv("SONAR_SCANNER_OPTS")
			newOpts := existingOpts + " " + p.Config.SonarOPS
			os.Setenv("SONAR_SCANNER_OPTS", newOpts)
		}

	} else if err == nil {
		// Configuration file exists, let sonar-scanner use it without additional parameters
		fmt.Println("Configuration file found. Using sonar-project.properties.")

		if len(p.Config.Host) >= 1 && p.Config.UseSonarConfigFileOverride {
			fmt.Println("OVERRIDING sonar.host.url=" + p.Config.Host)
			args = append(args, "-Dsonar.host.url="+p.Config.Host)
		}

		if len(p.Config.Token) >= 1 && p.Config.UseSonarConfigFileOverride {
			fmt.Println("OVERRIDING sonar.login=" + p.Config.Token)
			args = append(args, "-Dsonar.login="+p.Config.Token)
		}

		if len(p.Config.Key) >= 1 && p.Config.UseSonarConfigFileOverride {
			fmt.Println("OVERRIDING sonar.projectKey=" + p.Config.Key)
			args = append(args, "-Dsonar.projectKey="+p.Config.Key)
		}

	} else {
		// Error checking the file
		return fmt.Errorf("error checking configuration file: %v", err)
	}

	// Output sonar-scanner information
	fmt.Printf("\n\nStarting Plugin - Sonar Scanner Quality Gate Report\n")
	fmt.Printf("Developed by Diego Pereira\n")
	fmt.Printf("sonar Arguments: %v\n\n", args)

	status := ""
	taskFilePath := ".scannerwork/report-task.txt"
	if len(p.Config.Workspace) >= 1 {
		taskFilePath = p.Config.Workspace + "/.scannerwork/report-task.txt"
	}

	if p.Config.TaskId != "" || p.Config.SkipScan {
		fmt.Println("Skipping Scan...")
		fmt.Println("")
		fmt.Println("Waiting for quality gate validation...")
		fmt.Println("")
		status, err = PreFlightGetLatestTaskID(p.Config)
		if err != nil {
			fmt.Printf("\n\n==> Error getting the latest scanID\n\n")
			logConfigInfo("Error", err.Error())
			return err
		}
	} else {
		fmt.Println("Starting Analysis")
		fmt.Println("")
		cmd := exec.Command("sonar-scanner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("\n\n==> Error in Analysis\n\n")
			logConfigInfo("Error", err.Error())
			// return err
		}
		fmt.Println("")
		fmt.Println("==> Sonar Analysis Finished!")
		fmt.Println("")
		fmt.Println("")
		fmt.Println("Static Analysis Result:")
		fmt.Println("")
		fmt.Println("")

		cmd = exec.Command("cat", taskFilePath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Run command cat reportname failed")
			return err
		}

		fmt.Printf("\n\nParsing Results:\n\n")

		report, err := staticScan(&p, taskFilePath)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Unable to parse scan results!")
		}

		if p.Config.WaitQualityGate {
			logrus.WithFields(logrus.Fields{
				"job url": report.CeTaskURL,
			}).Info("Job url")
			fmt.Printf("\n\nWaiting Analysis to finish:\n\n")

			task, err := waitForSonarJob(report)
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"error": err,
				}).Fatal("Unable to get Job state")
				return err
			}

			fmt.Println("Waiting for quality gate validation...")
			fmt.Println("")

			status = getStatus(task, report)
		} else {
			fmt.Println("Delaying for quality gate validation...")
			fmt.Println("")
			status = "OK"
		}
	}
	fmt.Println("")
	fmt.Println("==> SONAR PROJECT DASHBOARD <==")
	fmt.Println("")
	fmt.Println(p.Config.Host + sonarDashStatic + p.Config.Key)
	fmt.Println("==> Harness CIE SonarQube Plugin with Quality Gateway <==")
	fmt.Println("")

	displayQualityGateStatus(status, p.Config.QualityEnabled == "true")

	if status != p.Config.Quality && p.Config.QualityEnabled == "true" {
		logrus.WithFields(logrus.Fields{
			"status": status,
		}).Info("QualityGate status failed. exiting...")
		os.Exit(p.Config.QualityGateErrorExitCode)
	}
	if status != p.Config.Quality && p.Config.QualityEnabled == "false" {
		logrus.WithFields(logrus.Fields{
			"status": status,
		}).Info("Quality Gate Status disabled")
	}
	if status == p.Config.Quality {
		logrus.WithFields(logrus.Fields{
			"status": status,
		}).Info("Quality Gate Status Success")
	}

	return nil
}

func displayQualityGateStatus(status string, qualityEnabled bool) {
	fmt.Println(lineBreak)
	fmt.Printf("|         QUALITY GATE STATUS REPORT           |\n")
	fmt.Println(lineBreak)

	if status == "OK" {
		fmt.Printf("|         STATUS              |      \033[32m%s\033[0m       |\n", status)
	} else {
		fmt.Printf("|         STATUS              |      \033[31m%s\033[0m       |\n", status)
	}

	fmt.Println(lineBreak)

	if qualityEnabled {
		fmt.Printf("|      QUALITY GATE ENABLED   |       \033[32mYES\033[0m       |\n")
	} else {
		fmt.Printf("|      QUALITY GATE ENABLED   |       \033[31mNO\033[0m        |\n")
	}

	fmt.Printf("----------------------------------------------\n\n")
	fmt.Println(lineBreak)
	fmt.Printf("|         Developed by: Diego Pereira          |\n")
	fmt.Println(lineBreak)
}

func staticScan(p *Plugin, taskFilePath string) (*SonarReport, error) {

	cmd := exec.Command("sed", "-e", "s/=/=\"/", "-e", "s/$/\"/", taskFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Run command sed failed")
		return nil, err
	}
	report := SonarReport{}
	err = toml.Unmarshal(output, &report)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Toml Unmarshal failed")
		return nil, err
	}

	return &report, nil
}

func getStatus(task *TaskResponse, report *SonarReport) string {

	qg_type := os.Getenv("PLUGIN_QG_TYPE")
	qg_projectKey := os.Getenv("PLUGIN_SONAR_KEY")

	var reportRequest url.Values

	if qg_type == "branch" {
		qg_branch := os.Getenv("PLUGIN_BRANCH")
		reportRequest = url.Values{
			"branch":     {qg_branch},
			"projectKey": {qg_projectKey},
		}
	} else if qg_type == "pullRequest" {
		qg_pr := os.Getenv("PLUGIN_PR_KEY")
		reportRequest = url.Values{
			"pullRequest": {qg_pr},
			"projectKey":  {qg_projectKey},
		}

	} else if qg_type == "projectKey" {
		reportRequest = url.Values{
			"projectKey": {qg_projectKey},
		}
	} else {
		reportRequest = url.Values{
			"analysisId": {task.Task.AnalysisID},
		}
	}

	sonarToken := os.Getenv("PLUGIN_SONAR_TOKEN")

	// First try with Basic Auth
	projectRequest, err := http.NewRequest("GET", report.ServerURL+"/api/qualitygates/project_status?"+reportRequest.Encode(), nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed get status")
	}
	fmt.Printf("==> Job Quality Gate Request:\n")
	fmt.Printf(report.ServerURL + "/api/qualitygates/project_status?" + reportRequest.Encode())
	fmt.Printf("\n")
	fmt.Printf("\n")
	projectRequest.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(sonarToken+":")))
	projectResponse, err := netClient.Do(projectRequest)

	if err != nil || projectResponse.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Info("Failed to get status with Basic Auth, retrying with Bearer token...")

		// Retry with Bearer token
		projectRequest.Header.Set("Authorization", "Bearer "+sonarToken)
		projectResponse, err = netClient.Do(projectRequest)

		if err != nil || projectResponse.StatusCode != http.StatusOK {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed to get status after retry with Bearer token")
		}
	}

	buf, _ := io.ReadAll(projectResponse.Body)
	fmt.Printf("==> Report Result:\n")
	fmt.Println(string(buf))
	fmt.Printf("\n")
	project := ProjectStatusResponse{}
	if err := json.Unmarshal(buf, &project); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed")
	}
	fmt.Printf("==> Report Result:\n")
	fmt.Println(string(buf))

	// JUNUT
	junitReport := ""
	junitReport = string(buf) // returns a string of what was written to it
	fmt.Println(lineBreak)
	fmt.Printf("|      SONAR SCAN + JUNIT EXPORTER PLUGIN      |\n")
	fmt.Print("----------------------------------------------\n\n\n")
	bytesReport := []byte(junitReport)
	var projectReport Project
	err = json.Unmarshal(bytesReport, &projectReport)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", projectReport)
	fmt.Printf("\n")
	result := ParseJunit(projectReport, qg_projectKey)
	file, _ := xml.MarshalIndent(result, "", " ")
	_ = ioutil.WriteFile("sonarResults.xml", file, 0644)

	fmt.Println(lineBreak)
	fmt.Printf("|  Harness Drone/CIE SonarQube Plugin Results  |\n")
	fmt.Print("----------------------------------------------\n\n\n")

	return project.ProjectStatus.Status
}

func getStatusID(taskIDOld string, sonarHost string, projectSlug string) (string, error) {
	// token := os.Getenv("PLUGIN_SONAR_TOKEN")

	taskID, err := GetLatestTaskID(sonarHost, projectSlug)
	if err != nil {
		fmt.Println("Failed to get the latest task ID:", err)
		return "", err
	}
	fmt.Println("Latest task ID:", taskID)

	reportRequest := url.Values{
		"analysisId": {taskID},
	}
	fmt.Printf("==> Job Status Request:\n")
	fmt.Printf(sonarHost + "/api/qualitygates/project_status?" + reportRequest.Encode())
	fmt.Printf("\n")
	fmt.Printf("\n")
	fmt.Printf("analysisId:" + taskID)
	fmt.Printf("\n")

	buf, err := GetProjectStatus(sonarHost, reportRequest.Encode(), projectSlug)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to get project status")
	}

	project := ProjectStatusResponse{}
	if err := json.Unmarshal(buf, &project); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed")
		return "", nil
	}

	fmt.Printf("==> Report Result:\n")
	fmt.Printf(string(buf))

	// JUNUT
	junitReport := ""
	junitReport = string(buf) // returns a string of what was written to it
	fmt.Printf("\n---------------------> JUNIT Exporter <---------------------\n")
	bytesReport := []byte(junitReport)
	var projectReport Project
	err = json.Unmarshal(bytesReport, &projectReport)
	if err != nil {
		panic(err)
	}
	qg_projectKey := os.Getenv("PLUGIN_SONAR_KEY")

	fmt.Printf("%+v", projectReport)
	fmt.Println("")
	result := ParseJunit(projectReport, qg_projectKey)
	file, _ := xml.MarshalIndent(result, "", " ")
	_ = os.WriteFile("sonarResults.xml", file, 0644)

	fmt.Println("")
	fmt.Printf("\n======> JUNIT Exporter <======\n")

	//JUNIT
	fmt.Printf("\n======> Harness Drone/CIE SonarQube Plugin <======\n\n====> Results:")

	return project.ProjectStatus.Status, nil
}

func getStatusV2(scanType string, scanValue string, sonarHost string, projectSlug string) (string, error) {
	// token := os.Getenv("PLUGIN_SONAR_TOKEN")

	fmt.Println("Searchng last analysis")

	var reportRequest url.Values

	if scanType == "branch" {
		fmt.Println("Searchng last analysis by branch")
		reportRequest = url.Values{
			"branch":     {scanValue},
			"projectKey": {projectSlug},
		}
	} else {
		fmt.Println("Searchng last analysis by pull request")
		reportRequest = url.Values{
			"pullRequest": {scanValue},
			"projectKey":  {projectSlug},
		}
	}

	fmt.Printf("==> Job Status Request:\n")
	fmt.Printf(sonarHost + "/api/qualitygates/project_status?" + reportRequest.Encode())
	fmt.Printf("\n")
	fmt.Printf("\n")
	fmt.Printf("scanType:" + scanType)
	fmt.Printf("scanValue:" + scanValue)
	fmt.Printf("\n")

	buf, err := GetProjectStatus(sonarHost, reportRequest.Encode(), projectSlug)

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to get project status")
	}

	project := ProjectStatusResponse{}
	if err := json.Unmarshal(buf, &project); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed")
		return "", nil
	}

	fmt.Printf("==> Report Result:\n")
	fmt.Printf(string(buf))

	// JUNUT
	junitReport := ""
	junitReport = string(buf) // returns a string of what was written to it
	fmt.Printf("\n---------------------> JUNIT Exporter <---------------------\n")
	bytesReport := []byte(junitReport)
	var projectReport Project
	err = json.Unmarshal(bytesReport, &projectReport)
	if err != nil {
		panic(err)
	}
	qg_projectKey := os.Getenv("PLUGIN_SONAR_KEY")

	fmt.Printf("%+v", projectReport)
	fmt.Printf("\n")
	result := ParseJunit(projectReport, qg_projectKey)
	file, _ := xml.MarshalIndent(result, "", " ")
	_ = os.WriteFile("sonarResults.xml", file, 0644)

	fmt.Printf("\n")
	fmt.Printf("\n======> JUNIT Exporter <======\n")

	//JUNIT
	fmt.Printf("\n======> Harness Drone/CIE SonarQube Plugin <======\n\n====> Results:")

	return project.ProjectStatus.Status, nil
}

func GetProjectStatus(sonarHost string, analysisId string, projectSlug string) ([]byte, error) {
	token := os.Getenv("PLUGIN_SONAR_TOKEN")
	fmt.Printf("\n")
	fmt.Printf("Getting project status: " + projectSlug + "\n" + analysisId)
	netClient := &http.Client{
		Timeout: time.Second * 10, // you can adjust the timeout
	}
	projectRequest, err := http.NewRequest("GET", sonarHost+"/api/qualitygates/project_status?"+analysisId, nil)
	if err != nil {
		return nil, err
	}
	fmt.Printf("URL:" + sonarHost + "/api/qualitygates/project_status?" + analysisId)

	fmt.Printf("\n")
	// fmt.Printf("Setting Authorization header:" + token)
	// Retry with the token encoded in base64
	encodedToken := base64.StdEncoding.EncodeToString([]byte(token + ":"))
	fmt.Println(basicAuth + encodedToken)
	projectRequest.Header.Set("Authorization", basicAuth+encodedToken)
	fmt.Printf("\n")
	// projectRequest.Header.Add("Authorization", basicAuth+token)
	projectResponse, err := netClient.Do(projectRequest)

	if err != nil {
		fmt.Printf("\n")
		fmt.Printf("NIL - Error getting project status, failed!")

		return nil, err

	}
	fmt.Printf("Response Code:" + projectResponse.Status)
	buf := []byte{}
	// if status code 401 try again with bearer token
	if projectResponse.StatusCode == 401 {
		bearer := "Bearer " + token
		projectBearerRequest, err := http.NewRequest("GET", sonarHost+"/api/qualitygates/project_status?"+analysisId, nil)
		if err != nil {
			fmt.Printf("\n")
			fmt.Printf("Error creating request")
			return nil, err
		}
		projectBearerRequest.Header.Add("Authorization", bearer)
		projectBearerResponse, err := netClient.Do(projectBearerRequest)
		if err != nil {
			fmt.Printf("\n")
			fmt.Printf("NIL - Error getting project status, trying again with bearer token...")
			return nil, err
		}
		fmt.Printf("Response Code with Bearer:" + projectBearerResponse.Status)
		if projectBearerResponse.StatusCode == 401 {
			fmt.Printf("\n")
			fmt.Printf("Error getting project status, trying again with bearer token...")

			return nil, fmt.Errorf("unauthorized to get project status")
		}
		bufResponse, err := ioutil.ReadAll(projectBearerResponse.Body)
		if err != nil {
			fmt.Printf("\n")
			fmt.Printf("Error parsing results...")
			return nil, err
		}
		buf = bufResponse
		defer projectBearerResponse.Body.Close()
		fmt.Printf("\n")
		// projectBearerResponse.Body.Close() // Always close the response body
	} else {
		fmt.Printf("\n")
		fmt.Printf("Requested project status, parsing results...")
		fmt.Printf("\n")
		bufBasicResponse, err := ioutil.ReadAll(projectResponse.Body)
		if err != nil {
			fmt.Printf("\n")
			fmt.Printf("Error parsing results...")
			return nil, err
		}
		buf = bufBasicResponse
	}

	fmt.Printf("\n")
	fmt.Printf("Quality Gate Results (JSON):")
	fmt.Printf("\n")
	fmt.Print(string(buf))
	fmt.Printf("\n")
	fmt.Printf("\n")
	defer projectResponse.Body.Close() // Always close the response body

	return buf, nil
}

func addBearerToken(req *http.Request, token string) {
	req.Header.Add("Authorization", "Bearer "+token)
}

func addBasicAuth(req *http.Request, token string) {
	req.SetBasicAuth(token, "")
}

func GetLatestTaskID(sonarHost string, projectSlug string) (string, error) {
	fmt.Printf("\nStarting Task ID Discovery\n")
	url := fmt.Sprintf("%s/api/project_analyses/search?project=%s&ps=1", sonarHost, projectSlug)
	fmt.Printf("URL: %s\n", url)

	taskRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("\nError to create request in Task discovery: %s\n", err.Error())
		return "", err
	}

	sonarToken := os.Getenv("PLUGIN_SONAR_TOKEN")
	// First, try with Bearer token
	addBearerToken(taskRequest, sonarToken)
	taskResponse, err := netClient.Do(taskRequest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed get sonar job status")
	}

	// If Forbidden, try with Basic Auth
	// if taskResponse.StatusCode == http.StatusForbidden {
	if taskResponse.StatusCode != http.StatusOK {
		fmt.Printf("\nRetrying with Basic Auth...\n")
		addBasicAuth(taskRequest, sonarToken)
		taskResponse, err = netClient.Do(taskRequest)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed get sonar job status")
		}
	}

	if taskResponse.StatusCode != http.StatusOK {
		if taskResponse.StatusCode == http.StatusUnauthorized {
			fmt.Printf("\nError in Task discovery: %s\n", "Invalid Credentials - your token is not valid")
		}
		return "", fmt.Errorf("HTTP request error. Status code: %d", taskResponse.StatusCode)
	}

	body, err := io.ReadAll(taskResponse.Body)
	if err != nil {
		fmt.Printf("\nError reading response body in Task discovery: %s\n", err.Error())
		return "", err
	}

	if len(body) == 0 {
		fmt.Printf("\nReceived empty response from server\n")
		return "", errors.New("received empty response from server")
	}

	bodyString := string(body)
	fmt.Printf("Response body: %s\n", bodyString)

	var data AnalysisResponse
	if err := json.Unmarshal(body, &data); err != nil {
		fmt.Printf("\nError unmarshalling response body: %s\n", err.Error())
		return "", err
	}

	if len(data.Analyses) == 0 {
		return "", fmt.Errorf("no analyses found for project %s", projectSlug)
	}

	return data.Analyses[0].Key, nil
}

func getSonarJobStatus(report *SonarReport) *TaskResponse {
	fmt.Printf("\n")
	fmt.Printf("==> Job Status Request:\n")
	fmt.Printf(report.ServerURL + "/api/ce/task?id=" + report.CeTaskID)
	fmt.Printf("\n")
	fmt.Printf("\n")

	taskRequest, err := http.NewRequest("GET", report.CeTaskURL, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to create request for Sonar job status")
	}

	sonarToken := os.Getenv("PLUGIN_SONAR_TOKEN")
	taskRequest.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(sonarToken+":")))

	taskResponse, err := netClient.Do(taskRequest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to get Sonar job status")
	}

	if taskResponse.StatusCode == http.StatusForbidden {
		fmt.Println("Basic Auth failed. Retrying with Bearer token...")
		taskRequest.Header.Set("Authorization", "Bearer "+sonarToken)
		taskResponse, err = netClient.Do(taskRequest)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed to get Sonar job status with Bearer token")
		}
	}

	buf, err := io.ReadAll(taskResponse.Body)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to read Sonar job status response body")
	}

	fmt.Printf("\n==> Job Status Response:\n")
	fmt.Println(string(buf))
	fmt.Printf("\n")

	task := TaskResponse{}

	fmt.Println(lineBreak2)
	fmt.Println("|  Report Result:                                                 |")
	fmt.Println(lineBreak2)
	fmt.Print(string(buf))
	fmt.Println(lineBreak2)
	json.Unmarshal(buf, &task)
	return &task
}

func waitForSonarJob(report *SonarReport) (*TaskResponse, error) {
	timeout := time.After(300 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	fmt.Println("Waiting for sonar job to finish...")
	for {
		select {
		case <-timeout:
			fmt.Println("Timed out waiting for sonar job to finish")
			return nil, errors.New("timed out")
		case <-tick:
			fmt.Println("Checking sonar job status...")
			job := getSonarJobStatus(report)
			if job.Task.Status == "SUCCESS" {
				fmt.Println("\033[32mSonar job finished successfully\033[0m")
				return job, nil
			}
			if job.Task.Status == "ERROR" {
				fmt.Println("Sonar job failed")
				return nil, errors.New("ERROR")
			}
		}
	}
}
