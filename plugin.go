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

	// basicAuth is the basic authentication string.
	basicAuth = "Basic "
)

type (
	Config struct {
		Key                       string
		Name                      string
		Host                      string
		Token                     string
		Version                   string
		Branch                    string
		Sources                   string
		Timeout                   string
		Inclusions                string
		Exclusions                string
		Level                     string
		ShowProfiling             string
		BranchAnalysis            bool
		UsingProperties           bool
		Binaries                  string
		Quality                   string
		QualityEnabled            string
		QualityTimeout            string
		ArtifactFile              string
		JavascitptIcovReport      string
		JavaCoveragePlugin        string
		JacocoReportPath          string
		SSLKeyStorePassword       string
		CacertsLocation           string
		JunitReportPaths          string
		SourceEncoding            string
		SonarTests                string
		JavaTest                  string
		PRKey                     string
		PRBranch                  string
		PRBase                    string
		CoverageExclusion         string
		JavaSource                string
		JavaLibraries             string
		SurefireReportsPath       string
		TypescriptLcovReportPaths string
		Verbose                   string
		CustomJvmParams           string
		TaskId                    string
		SkipScan                  bool
		WaitQualityGate           bool
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
			ID            string `json:"id"`
			Type          string `json:"type"`
			ComponentID   string `json:"componentId"`
			ComponentKey  string `json:"componentKey"`
			ComponentName string `json:"componentName"`
			AnalysisID    string `json:"analysisId"`
			Status        string `json:"status"`
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

	// type Period struct {
	//     Index     int    `json:"index"`
	//     Mode      string `json:"mode"`
	//     Date      string `json:"date"`
	//     Parameter string `json:"parameter,omitempty"` // this might not always be present
	// }

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

const lineBreak = "----------------------------------------------"

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
	// file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	fmt.Println("Error opening/creating .env file:", err)
	// 	// return
	// }

	// for key, value := range vars {
	// 	fmt.Println("Writing to .env file:", key, value)
	// 	_, err = file.WriteString(fmt.Sprintf("%s=%s\n", key, value))
	// 	if err != nil {
	// 		fmt.Println("Error writing to .env file:", err)
	// 		// return
	// 	}
	// }

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

	dashboardLink := os.Getenv("PLUGIN_SONAR_HOST") + sonarDashStatic + os.Getenv("PLUGIN_SONAR_NAME")
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

func (p Plugin) Exec() error {
	// Initial values
	args := []string{
		"-Dsonar.host.url=" + p.Config.Host,
		"-Dsonar.login=" + p.Config.Token,
	}

	// Map of potential configurations
	configurations := map[string]string{
		"-Dsonar.projectKey":                        p.Config.Key,
		"-Dsonar.projectName":                       p.Config.Name,
		"-Dsonar.projectVersion":                    p.Config.Version,
		"-Dsonar.sources":                           p.Config.Sources,
		"-Dsonar.ws.timeout":                        p.Config.Timeout,
		"-Dsonar.inclusions":                        p.Config.Inclusions,
		"-Dsonar.exclusions":                        p.Config.Exclusions,
		"-Dsonar.log.level":                         p.Config.Level,
		"-Dsonar.showProfiling":                     p.Config.ShowProfiling,
		"-Dsonar.java.binaries":                     p.Config.Binaries,
		"-Dsonar.branch.name":                       p.Config.Branch,
		"-Dsonar.qualitygate.wait":                  strconv.FormatBool(p.Config.WaitQualityGate),
		"-Dsonar.qualitygate.timeout":               p.Config.QualityTimeout,
		"-Dsonar.javascript.lcov.reportPaths":       p.Config.JavascitptIcovReport,
		"-Dsonar.coverage.jacoco.xmlReportPaths":    p.Config.JacocoReportPath,
		"-Dsonar.java.coveragePlugin":               p.Config.JavaCoveragePlugin,
		"-Dsonar.junit.reportPaths":                 p.Config.JunitReportPaths,
		"-Dsonar.sourceEncoding":                    p.Config.SourceEncoding,
		"-Dsonar.tests":                             p.Config.SonarTests,
		"-Dsonar.java.test.binaries":                p.Config.JavaTest,
		"-Dsonar.coverage.exclusions":               p.Config.CoverageExclusion,
		"-Dsonar.java.source":                       p.Config.JavaSource,
		"-Dsonar.java.libraries":                    p.Config.JavaLibraries,
		"-Dsonar.surefire.reportsPath":              p.Config.SurefireReportsPath,
		"-Dsonar.sonar.typescript.lcov.reportPaths": p.Config.TypescriptLcovReportPaths,
		"-Dsonar.verbose":                           p.Config.Verbose,
		"-Dsonar.pullrequest.key":                   p.Config.PRKey,
		"-Dsonar.pullrequest.branch":                p.Config.PRBranch,
		"-Dsonar.pullrequest.base":                  p.Config.PRBase,
		"-Djavax.net.ssl.trustStorePassword":        p.Config.SSLKeyStorePassword,
		"-Djavax.net.ssl.trustStore":                p.Config.CacertsLocation,
	}

	// Loop over the configurations and add to args if they exist
	for config, value := range configurations {
		if len(value) >= 1 {
			args = append(args, config+"="+value)
		}
	}

	// Special conditions
	if len(p.Config.Verbose) >= 1 {
		args = append(args, "-X")
	}

	if !p.Config.UsingProperties {
		args = append(args, "-Dsonar.scm.provider=git")
	}

	if len(p.Config.CustomJvmParams) >= 1 {
		params := strings.Split(p.Config.CustomJvmParams, ",")
		args = append(args, params...)
	}

	// Assuming your struct has a print or log method
	if len(p.Config.JacocoReportPath) >= 1 {
		fmt.Printf("\n\n==> Sonar Java Plugin Jacoco configured!\n\n")
		fmt.Printf("\n\n==> -Dsonar.coverage.jacoco.xmlReportPaths=" + p.Config.JacocoReportPath + "\n\n")
	}

	if len(p.Config.JavaCoveragePlugin) >= 1 {
		fmt.Printf("\n\n==> Sonar Java Plugin Jacoco Path configured!\n\n")
	}

	// args := []string{
	// 	"-Dsonar.host.url=" + p.Config.Host,
	// 	"-Dsonar.login=" + p.Config.Token,
	// }
	// projectFinalKey := p.Config.Key

	// if len(p.Config.Verbose) >= 1 {
	// 	args = append(args, "-X")
	// }

	// if !p.Config.UsingProperties {
	// 	argsParameter := []string{
	// 		"-Dsonar.projectKey=" + projectFinalKey,
	// 		"-Dsonar.projectName=" + p.Config.Name,
	// 		"-Dsonar.projectVersion=" + p.Config.Version,
	// 		"-Dsonar.sources=" + p.Config.Sources,
	// 		"-Dsonar.ws.timeout=" + p.Config.Timeout,
	// 		"-Dsonar.inclusions=" + p.Config.Inclusions,
	// 		"-Dsonar.exclusions=" + p.Config.Exclusions,
	// 		"-Dsonar.log.level=" + p.Config.Level,
	// 		"-Dsonar.showProfiling=" + p.Config.ShowProfiling,
	// 		"-Dsonar.scm.provider=git",
	// 		"-Dsonar.java.binaries=" + p.Config.Binaries,
	// 	}
	// 	args = append(args, argsParameter...)
	// }
	// if p.Config.BranchAnalysis {
	// 	args = append(args, "-Dsonar.branch.name="+p.Config.Branch)
	// }
	// if p.Config.QualityEnabled == "true" {
	// 	args = append(args, "-Dsonar.qualitygate.wait="+p.Config.QualityEnabled)
	// 	args = append(args, "-Dsonar.qualitygate.timeout="+p.Config.QualityTimeout)
	// }
	// if len(p.Config.JavascitptIcovReport) >= 1 {
	// 	args = append(args, "-Dsonar.javascript.lcov.reportPaths="+p.Config.JavascitptIcovReport)
	// }
	// if len(p.Config.JacocoReportPath) >= 1 {
	// 	args = append(args, "-Dsonar.coverage.jacoco.xmlReportPaths="+p.Config.JacocoReportPath)
	// 	fmt.Printf("\n\n==> Sonar Java Plugin Jacoco configured!\n\n")
	// 	fmt.Printf("\n\n==> -Dsonar.coverage.jacoco.xmlReportPaths=" + p.Config.JacocoReportPath + "\n\n")
	// }
	// if len(p.Config.JavaCoveragePlugin) >= 1 {
	// 	args = append(args, "-Dsonar.java.coveragePlugin="+p.Config.JavaCoveragePlugin)
	// 	fmt.Printf("\n\n==> Sonar Java Plugin Jacoco Path configured!\n\n")
	// }
	// if len(p.Config.JunitReportPaths) >= 1 {
	// 	args = append(args, "-Dsonar.junit.reportPaths="+p.Config.JunitReportPaths)
	// }
	// if len(p.Config.SourceEncoding) >= 1 {
	// 	args = append(args, "-Dsonar.sourceEncoding="+p.Config.SourceEncoding)
	// }
	// if len(p.Config.SonarTests) >= 1 {
	// 	args = append(args, "-Dsonar.tests="+p.Config.SonarTests)
	// }
	// if len(p.Config.JavaTest) >= 1 {
	// 	args = append(args, "-Dsonar.java.test.binaries="+p.Config.JavaTest)
	// }
	// if len(p.Config.CoverageExclusion) >= 1 {
	// 	args = append(args, "-Dsonar.coverage.exclusions="+p.Config.CoverageExclusion)
	// }
	// if len(p.Config.JavaSource) >= 1 {
	// 	args = append(args, "-Dsonar.java.source="+p.Config.JavaSource)
	// }
	// if len(p.Config.JavaLibraries) >= 1 {
	// 	args = append(args, "-Dsonar.java.libraries="+p.Config.JavaLibraries)
	// }
	// if len(p.Config.SurefireReportsPath) >= 1 {
	// 	args = append(args, "-Dsonar.surefire.reportsPath="+p.Config.SurefireReportsPath)
	// }
	// if len(p.Config.TypescriptLcovReportPaths) >= 1 {
	// 	args = append(args, "-Dsonar.sonar.typescript.lcov.reportPaths="+p.Config.TypescriptLcovReportPaths)
	// }
	// if len(p.Config.Verbose) >= 1 {
	// 	args = append(args, "-Dsonar.verbose="+p.Config.Verbose)
	// }

	// if len(p.Config.CustomJvmParams) >= 1 {

	// 	params := strings.Split(p.Config.CustomJvmParams, ",")

	// 	for _, param := range params {
	// 		//fmt.Println(i, param)
	// 		args = append(args, param)
	// 	}

	// }

	// if len(p.Config.PRKey) >= 1 {
	// 	args = append(args, "-Dsonar.pullrequest.key="+p.Config.PRKey)
	// }

	// if len(p.Config.PRBranch) >= 1 {
	// 	args = append(args, "-Dsonar.pullrequest.branch="+p.Config.PRBranch)
	// }

	// if len(p.Config.PRBase) >= 1 {
	// 	args = append(args, "-Dsonar.pullrequest.base="+p.Config.PRBase)
	// }

	// if len(p.Config.SSLKeyStorePassword) >= 1 {
	// 	args = append(args, "-Djavax.net.ssl.trustStorePassword="+p.Config.SSLKeyStorePassword)
	// }

	// if len(p.Config.CacertsLocation) >= 1 {
	// 	args = append(args, "-Djavax.net.ssl.trustStore="+p.Config.CacertsLocation)
	// }

	os.Setenv("SONAR_USER_HOME", ".sonar")

	fmt.Printf("\n\n")
	fmt.Printf("Starting Plugin - Sonar Scanner Quality Gate Report")
	fmt.Printf("\n")
	fmt.Printf("Developed by Diego Pereira")
	fmt.Printf("\n")
	fmt.Printf("sonar Arguments:")
	fmt.Printf("%v", args)
	fmt.Printf("\n")
	fmt.Printf("\n")

	status := ""

	if p.Config.TaskId != "" || p.Config.SkipScan {
		fmt.Printf("Skipping Scan...")
		fmt.Printf("\n")
		fmt.Printf("\n")
		fmt.Printf("#######################################\n")
		fmt.Printf("Waiting for quality gate validation...\n")
		fmt.Printf("#######################################\n")
		statusID, err := getStatusID(p.Config.TaskId, p.Config.Host, p.Config.Key)
		if err != nil {
			fmt.Printf("\n\n==> Error getting the latest scanID\n\n")
			fmt.Printf("Error: %s", err.Error())
			return err
		}
		status = statusID
	} else {
		fmt.Printf("Starting Analisys")
		fmt.Printf("\n")
		cmd := exec.Command("sonar-scanner", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("\n\n==> Error in Analysis\n\n")
			fmt.Printf("Error: %s", err.Error())
			//return err
		}
		fmt.Printf("\n==> Sonar Analysis Finished!\n\n")
		fmt.Printf("\n\nStatic Analysis Result:\n\n")

		cmd = exec.Command("cat", ".scannerwork/report-task.txt")

		cmd.Stdout = os.Stdout

		cmd.Stderr = os.Stderr
		fmt.Printf("\n")
		fmt.Printf("#######################################\n")
		fmt.Printf("==> Report Result:\n")
		fmt.Printf("#######################################\n")
		fmt.Printf("\n")
		err = cmd.Run()

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Run command cat reportname failed")
			return err
		}

		fmt.Printf("\n\nParsing Results:\n\n")
		fmt.Printf("\n")

		report, err := staticScan(&p)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Unable to parse scan results!")
		}
		logrus.WithFields(logrus.Fields{
			"job url": report.CeTaskURL,
		}).Info("Job url")
		fmt.Printf("\n")
		fmt.Printf("\n\nWaiting Analysis to finish:\n\n")
		fmt.Printf("\n")

		task, err := waitForSonarJob(report)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Unable to get Job state")
			return err
		}

		fmt.Printf("\n")
		fmt.Printf("#######################################\n")
		fmt.Printf("Waiting for quality gate validation...\n")
		fmt.Printf("#######################################\n")
		fmt.Printf("\n")

		status = getStatus(task, report)
	}

	fmt.Printf("\n")
	fmt.Printf("==> SONAR PROJECT DASHBOARD <==\n")
	fmt.Printf(p.Config.Host)
	fmt.Printf(sonarDashStatic)
	fmt.Printf(p.Config.Name)
	fmt.Printf("\n==> Harness CIE SonarQube Plugin with Quality Gateway <==\n\n")
	// "Docker", p.Config.ArtifactFile, (p.Config.Host + sonarDashStatic + p.Config.Name), "Sonar", "Harness Sonar Plugin", []string{"Diego", "latest"})

	displayQualityGateStatus(status, p.Config.QualityEnabled == "true")

	if status != p.Config.Quality && p.Config.QualityEnabled == "true" {
		// fmt.Printf("\n==> QUALITY ENABLED ENALED  - set quality_gate_enabled as false to disable qg\n")
		logrus.WithFields(logrus.Fields{
			"status": status,
		}).Fatal("QualityGate status failed")
	}
	if status != p.Config.Quality && p.Config.QualityEnabled == "false" {
		// fmt.Printf("\n==> QUALITY GATEWAY DISABLED\n")
		// fmt.Printf("\n==> FAILED <==\n")
		logrus.WithFields(logrus.Fields{
			"status": status,
		}).Info("Quality Gate Status FAILED")
	}
	if status == p.Config.Quality {
		// fmt.Printf("\n==> QUALITY GATEWAY ENALED \n")
		// fmt.Printf("\n==> PASSED <==\n")
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

func staticScan(p *Plugin) (*SonarReport, error) {

	cmd := exec.Command("sed", "-e", "s/=/=\"/", "-e", "s/$/\"/", ".scannerwork/report-task.txt")
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
	reportRequest := url.Values{
		"analysisId": {task.Task.AnalysisID},
	}
	sonarToken := os.Getenv("PLUGIN_SONAR_TOKEN")
	projectRequest, err := http.NewRequest("GET", report.ServerURL+"/api/qualitygates/project_status?"+reportRequest.Encode(), nil)
	projectRequest.Header.Add("Authorization", basicAuth+sonarToken)
	projectResponse, err := netClient.Do(projectRequest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Info("Failed to get status, retrying with encoded token...")

		// Retry with the token encoded in base64
		encodedToken := base64.StdEncoding.EncodeToString([]byte(sonarToken))
		projectRequest.Header.Set("Authorization", "Basic "+encodedToken)
		projectResponse, err = netClient.Do(projectRequest)

		if err != nil {
			logrus.WithFields(logrus.Fields{
				"error": err,
			}).Fatal("Failed to get status after retry")
		}
	}
	buf, _ := ioutil.ReadAll(projectResponse.Body)
	project := ProjectStatusResponse{}
	if err := json.Unmarshal(buf, &project); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed")
	}
	fmt.Printf("==> Report Result:\n")
	fmt.Printf(string(buf))

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
	result := ParseJunit(projectReport, "BankingApp")
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

	// projectRequest, err := http.NewRequest("GET", sonarHost+"/api/qualitygates/project_status?"+reportRequest.Encode(), nil)
	// projectRequest.Header.Add("Authorization", basicAuth+token)
	// projectResponse, err := netClient.Do(projectRequest)
	// if err != nil {
	// 	logrus.WithFields(logrus.Fields{
	// 		"error": err,
	// 	}).Fatal("Failed get status")
	// 	return "", err
	// }
	// buf, _ := ioutil.ReadAll(projectResponse.Body)
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

	fmt.Printf("%+v", projectReport)
	fmt.Printf("\n")
	result := ParseJunit(projectReport, "BankingApp")
	file, _ := xml.MarshalIndent(result, "", " ")
	_ = ioutil.WriteFile("sonarResults.xml", file, 0644)

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

func GetLatestTaskID(sonarHost string, projectSlug string) (string, error) {
	fmt.Printf("\nStarting Task ID Discovery\n")
	url := fmt.Sprintf("%s/api/project_analyses/search?project=%s&ps=1", sonarHost, projectSlug)
	fmt.Printf("URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("\nError to create request in Task discovery: %s\n", err.Error())
		return "", err
	}

	sonarToken := os.Getenv("PLUGIN_SONAR_TOKEN")
	req.SetBasicAuth(sonarToken, "")
	resp, err := netClient.Do(req)
	if err != nil {
		fmt.Printf("\nRequest Error in Task discovery: %s\n", err.Error())
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		fmt.Printf("\nError in Task discovery: %s\n", "Check your token permission - probably it does not have 'Browse' permission on the project")
		fmt.Printf("Retrying with encoded token...\n")

		encodedToken := base64.StdEncoding.EncodeToString([]byte(sonarToken))
		req.Header.Add("Authorization", basicAuth+encodedToken)
		fmt.Printf("Token encoded: %s\n", encodedToken)
		req.SetBasicAuth(encodedToken, "")
		resp, err = netClient.Do(req)
		if err != nil {
			fmt.Printf("\nRequest Error in Task discovery after retry: %s\n", err.Error())
			return "", err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Printf("\nError in Task discovery: %s\n", "Invalid Credentials - your token is not valid")
		}
		return "", fmt.Errorf("HTTP request error. Status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
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
		}).Fatal("Failed get sonar job status")
	}
	taskRequest.Header.Add("Authorization", basicAuth+os.Getenv("PLUGIN_SONAR_TOKEN"))
	taskResponse, err := netClient.Do(taskRequest)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed get sonar job status")
	}
	buf, err := io.ReadAll(taskResponse.Body)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Failed to read sonar job status response body")
	}
	task := TaskResponse{}
	fmt.Println("|----------------------------------------------------------------|")
	fmt.Println("|  Report Result:                                                 |")
	fmt.Println("|----------------------------------------------------------------|")
	fmt.Print(string(buf))
	fmt.Println("|----------------------------------------------------------------|")
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
