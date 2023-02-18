package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type flagsArray []string

func (i *flagsArray) String() string {
	return "my string representation"
}

func (s *flagsArray) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

type ResponseBody struct {
	Message string `json:"message"`
}

type Project struct {
	ProjectFileVersion string `json:"projectFileVersion"`
	StringIndexType    string `json:"stringIndexType"`
	Metadata           struct {
		ProjectKind string `json:"projectKind"`
		Settings    struct {
			ConfidenceThreshold float64 `json:"confidenceThreshold"`
		} `json:"settings"`
		ProjectName  string `json:"projectName"`
		Multilingual bool   `json:"multilingual"`
		Description  string `json:"description"`
		Language     string `json:"language"`
	} `json:"metadata"`
	Assets struct {
		ProjectKind string `json:"projectKind"`
		Intents     []struct {
			Category string `json:"category"`
		} `json:"intents"`
		Entities []struct {
			Category           string `json:"category"`
			CompositionSetting string `json:"compositionSetting"`
			Prebuilts          []struct {
				Category string `json:"category"`
			} `json:"prebuilts,omitempty"`
		} `json:"entities"`
		Utterances []struct {
			Text     string `json:"text"`
			Language string `json:"language,omitempty"`
			Intent   string `json:"intent"`
			Entities []struct {
				Category string `json:"category"`
				Offset   int    `json:"offset"`
				Length   int    `json:"length"`
			} `json:"entities,omitempty"`
			Dataset string `json:"dataset,omitempty"`
		} `json:"utterances"`
	} `json:"assets"`
}

type Response struct {
	ResultURL string `json:"resultUrl"`
	// add other fields here
}

var projects flagsArray
var proj_json_array []Project

func main() {
          
        var uat_ep, vnext_ep, uat_key, vnext_key string

	flag.Var(&projects, "projects", "list of clu projects to migrate from vnext to uat.")
	flag.StringVar(&uat_ep, "uat-language-endpoint", "", "the endpoint of the UAT language service.")
	flag.StringVar(&vnext_ep, "vnext-language-endpoint", "", "the endpoint of the Vnext language service.")
	flag.StringVar(&uat_key, "uat-key", "", "the ocp-subscription-key for uat language service.")
	flag.StringVar(&vnext_key, "vnext-key", "", "the ocp-subscription-key for vnext language service.")

	flag.Parse()

	for _, project := range projects {

		proj_json_array = append(proj_json_array, ExportCLU(project, vnext_ep, vnext_key))

	}

	uat_proj_names := []string{"import-test-1", "import-test-2", "import-test-3", "import-test-4"}

	for index, project := range proj_json_array {

		ImportCLU(project, uat_ep, uat_key, uat_proj_names[index])

	}
}

func ExportCLU(project_name string, vnext_ep string, vnext_key string) Project {

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req_uri := fmt.Sprintf("%slanguage/authoring/analyze-conversations/projects/%s/:export?stringIndexType=Utf16CodeUnit&api-version=2022-05-01", vnext_ep, project_name)

	fmt.Println("\n\n################################################################################")
	fmt.Printf("Exporting CLU project \"%s\" from vnext", project_name)
	fmt.Println("\n################################################################################")

	req, err := http.NewRequest("POST", req_uri, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req.Header.Add("Ocp-Apim-Subscription-Key", vnext_key)

	response, err := client.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req_uri = response.Header["Operation-Location"][0]

	time.Sleep(5 * time.Second)

	req, err = http.NewRequest("GET", req_uri, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req.Header.Add("Ocp-Apim-Subscription-Key", vnext_key)

	response, err = client.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var resp Response

	err = json.Unmarshal(body, &resp)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req_uri = resp.ResultURL

	req, err = http.NewRequest("GET", req_uri, nil)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req.Header.Add("Ocp-Apim-Subscription-Key", vnext_key)

	response, err = client.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	body, err = ioutil.ReadAll(response.Body)

	var project_schema Project

	err = json.Unmarshal([]byte(body), &project_schema)

	defer response.Body.Close()

	return project_schema
}

func ImportCLU(project Project, uat_ep string, uat_key string, project_name string) {

	project.Metadata.ProjectName = project_name

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	json_body, err := json.Marshal(project)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n\n################################################################################")
	fmt.Printf("Importing CLU project \"%s\" to uat", project_name)
	fmt.Println("\n################################################################################")

	req_uri := fmt.Sprintf("%slanguage/authoring/analyze-conversations/projects/%s/:import?api-version=2022-05-01", uat_ep, project_name)

	req, err := http.NewRequest("POST", req_uri, bytes.NewBuffer(json_body))

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Add("Ocp-Apim-Subscription-Key", uat_key)

	response, err := client.Do(req)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	defer response.Body.Close()

}

//func Train()  {}
//func Deploy() {}
