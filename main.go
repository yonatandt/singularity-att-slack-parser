package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twinj/uuid"
)

const (
	LOG_FILE_PATH = "../logs/message.log"
)

// Feature Flag Change Type
const (
	Enable      = "Enable"
	Disable     = "Disable"
	Complicated = "It's complicated..."
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World!")
	// handle post request:
	if r.Method == "POST" {
		fmt.Println("POST request")
		// print the post data:
		r.ParseForm()
		fmt.Println("Form data: ", r.Form)

		// get "text" param from post data:
		text := r.PostFormValue("text")
		fmt.Println("Message (before marshalling): ", text)

		// parse the post data to a Message struct:
		message := parseJSONToMessage(text)
		fmt.Println("Message (after marshalling): ", message)

		var structuredMessage StructuredMessage

		switch message.Type {
		case "deploy":
			// deploy := message.ToDeploy()
			// fmt.Println("Deploy: ", deploy) // for debugging
			// structuredMessage = deploy.structure()
			// fmt.Printf("DeployStructured: %+v\n", structuredMessage) // for debugging
			structuredMessage = message.ToDeploy2()
		case "ff-change":
			structuredMessage = message.ToFeatureFlagChange()
			fmt.Println("FeatureFlagChange: ", structuredMessage) // for debugging
		default:
			fmt.Println("Unknown message type: ", message.Type)
			return
		}

		fmt.Printf("FinalStructured: %+v\n", structuredMessage)
		// write the DeployStructured struct to a file:
		write_structured_json_to_file(&structuredMessage, LOG_FILE_PATH)
	}
}

func replaceCommasWithQuotationMarks(s string) string {
	s = strings.Replace(s, "”", "\"", -1)
	s = strings.Replace(s, "“", "\"", -1)
	return s
}

func parseAndFixQueryInMessage(s string) string {
	s = strings.Replace(s, "\n", " ", -1)
	arr := strings.Split(s, "&lt;query&gt;")
	if len(arr) < 2 {
		return s
	}
	arr[1] = strings.Replace(arr[1], "\"", "\\\"", -1)
	return strings.Join(arr, "")
}

type Message struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	// Deploy Related Fields:
	PullRequest   string `json:"pull_request,omitempty"`
	Components    string `json:"affected_services,omitempty"`
	WithMigration string `json:"includes_migration,omitempty"`
	// Feature Flag Change Related Fields:
	FFChangeType   string `json:"change_type,omitempty"`
	FeatureName    string `json:"feature_name,omitempty"`
	IsForAllOrgs   string `json:"is_for_all_orgs,omitempty"`
	Orgs           string `json:"orgs,omitempty"`
	AdditionalInfo string `json:"additional_info,omitempty"`
	Query          string `json:"query,omitempty"`
}

type Deploy struct {
	ID            string   `json:"id"`
	Type          string   `json:"event_type"`
	Subtype       string   `json:"subtype"`
	Description   string   `json:"header"`
	RepoName      string   `json:"repo_name",omitempty`
	PullRequest   string   `json:"pull_request,omitempty"`
	Components    []string `json:"affected_services,omitempty"`
	WithMigration bool     `json:"migration_files,omitempty"`
	Timestamps    string   `json:"timestamps,omitempty"`
}

type FeatureFlagChange struct {
	ID              string           `json:"id"`
	Type            string           `json:"event_type"`
	Subtype         string           `json:"subtype"`
	Description     string           `json:"header"`
	Subject         Subject          `json:"subject"`
	RelatedSubjects []RelatedSubject `json:"related_subjects"`
	Timestamps      string           `json:"timestamps"`
}

type DeployStructured struct {
	ID              string           `json:"id"`
	Type            string           `json:"event_type"`
	Subtype         string           `json:"subtype"`
	Description     string           `json:"header"`
	Subject         Subject          `json:"subject"`
	RelatedSubjects []RelatedSubject `json:"related_subjects"`
	Timestamps      string           `json:"timestamps"`
}

type StructuredMessage struct {
	ID              string           `json:"id"`
	Type            string           `json:"event_type"`
	Subtype         string           `json:"subtype"`
	Description     string           `json:"header"`
	Subject         Subject          `json:"subject"`
	RelatedSubjects []RelatedSubject `json:"related_subjects"`
	Timestamps      string           `json:"timestamps"`
}

type Subject struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type RelatedSubject struct {
	Type        string   `json:"type"`
	StringValue string   `json:"string_value,omitempty"`
	ArrayValue  []string `json:"array_value,omitempty"`
}

func parseJSONToMessage(jsonText string) Message {
	jsonText = parseAndFixQueryInMessage(replaceCommasWithQuotationMarks(jsonText))
	fmt.Println("Message (after replacing commas with quotation marks): ", jsonText) // for debugging

	var message Message
	err := json.Unmarshal([]byte(jsonText), &message)
	if err != nil {
		log.Printf("Error parsing JSON: %s", err)
	}
	return message
}

func (m *Message) ToDeploy() Deploy {
	var deploy Deploy
	deploy.ID = uuid.NewV4().String()
	deploy.Type = m.Type
	deploy.Subtype = "merge"
	deploy.Description = m.Description
	deploy.PullRequest = removeFirstCharAndLastChar(m.PullRequest)
	deploy.RepoName = getRepoNameFromGithubPrURL(deploy.PullRequest)
	deploy.WithMigration = translateToBoolean(m.WithMigration)
	// split by , and remove spaces:
	if m.Components != "" {
		deploy.Components = strings.Split(m.Components, ",")
	}
	deploy.Timestamps = time.Now().Format(time.RFC3339)
	return deploy
}

func (deploy *Deploy) structure() StructuredMessage {
	var deployStructured StructuredMessage
	deployStructured.ID = deploy.ID
	deployStructured.Type = deploy.Type
	deployStructured.Subtype = deploy.Subtype
	deployStructured.Description = deploy.Description
	deployStructured.Timestamps = deploy.Timestamps
	// create subject:
	deployStructured.Subject = Subject{Type: "repo", Name: deploy.RepoName}
	// create related subjects:
	relatedSubjects := make([]RelatedSubject, 0)
	if deploy.PullRequest != "" {
		relatedSubject := RelatedSubject{Type: "pull_request", StringValue: deploy.PullRequest}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	if len(deploy.Components) > 0 {
		relatedSubject := RelatedSubject{Type: "components", ArrayValue: deploy.Components}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "has_migration", StringValue: strconv.FormatBool(deploy.WithMigration)}) //TODO: fix this!
	deployStructured.RelatedSubjects = relatedSubjects
	return deployStructured
}

func (m *Message) ToDeploy2() StructuredMessage {
	var deploy StructuredMessage
	deploy.ID = uuid.NewV4().String()
	deploy.Type = m.Type
	deploy.Subtype = "merge"
	deploy.Description = m.Description
	deploy.Timestamps = time.Now().Format(time.RFC3339)

	pr := removeFirstCharAndLastChar(m.PullRequest)

	deploy.Subject = Subject{Type: "repo", Name: getRepoNameFromGithubPrURL(pr)}

	relatedSubjects := make([]RelatedSubject, 0)
	if m.PullRequest != "" {
		relatedSubject := RelatedSubject{Type: "pull_request", StringValue: pr}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	if m.Components != "" {
		relatedSubject := RelatedSubject{Type: "components", ArrayValue: strings.Split(m.Components, ",")}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "has_migration", StringValue: strconv.FormatBool(translateToBoolean(m.WithMigration))}) //TODO: fix this!
	deploy.RelatedSubjects = relatedSubjects

	return deploy
}

func (m *Message) ToFeatureFlagChange() StructuredMessage {
	var featureFlagChange StructuredMessage
	featureFlagChange.ID = uuid.NewV4().String()
	featureFlagChange.Type = m.Type
	featureFlagChange.Subtype = m.FFChangeType
	featureFlagChange.Description = m.BuildFFChangeDescription()
	featureFlagChange.Timestamps = time.Now().Format(time.RFC3339)

	featureFlagChange.Subject = Subject{Type: "feature_flag", Name: m.FeatureName}

	relatedSubjects := []RelatedSubject{
		{Type: "is_for_all_orgs", StringValue: strconv.FormatBool(translateToBoolean(m.IsForAllOrgs))},
		{Type: "query", StringValue: strings.ReplaceAll(m.Query, "\n", " ")},
	}

	if m.Orgs != "" {
		orgs_array := strings.Split(m.Orgs, ",")
		relatedSubject := RelatedSubject{Type: "orgs", ArrayValue: orgs_array}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	if m.AdditionalInfo != "" {
		relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "additional_info", StringValue: m.AdditionalInfo})
	}
	featureFlagChange.RelatedSubjects = relatedSubjects

	return featureFlagChange
}

func (m *Message) BuildFFChangeDescription() string {
	var description string

	action := "changed"
	switch m.FFChangeType {
	case Enable:
		action = "enabled"
	case Disable:
		action = "disabled"
	}

	if m.IsForAllOrgs == "true" {
		description = "Feature Flag " + m.FeatureName + " has been " + action + " for all orgs."
	} else {
		description = "Feature Flag " + m.FeatureName + " has been " + action + " for orgs: " + m.Orgs
	}
	return description
}

func write_structured_json_to_file(sm *StructuredMessage, filepath string) {
	// convert deploy to json:
	json, err := json.Marshal(sm)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
	}
	fmt.Println("Deploy (after marshalling): ", string(json)) // for debugging
	// append JSON to file
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file: %s", err)
	}
	defer f.Close()
	if _, err := f.WriteString(string(json) + "\n"); err != nil {
		log.Printf("Error writing to file: %s", err)
	}
}

func main() {
	http.HandleFunc("/deploy", handler)
	log.Println("Server starting on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func removeFirstCharAndLastChar(str string) string {
	if len(str) < 2 {
		return str
	}
	str = strings.Split(str, "|")[0]
	return str[1 : len(str)-1]
}

func translateToBoolean(str string) bool {
	return str == "Yes"
}

func getRepoNameFromGithubPrURL(url string) string {
	splitArr := strings.Split(url, "/")
	if len(splitArr) < 5 {
		return "attribution"
	}
	return fmt.Sprintf("%s/%s", splitArr[3], splitArr[4])
}
