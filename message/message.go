package message

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/twinj/uuid"
	"github.com/yonatandt/singularity-slack-reader/utils"
)

// Feature Flag Change Type
const (
	Enable      = "Enable"
	Disable     = "Disable"
	Complicated = "It's complicated..."
)

type Message struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	// Deploy Related Fields:
	PullRequest   string `json:"pull_request,omitempty"`
	Components    string `json:"affected_services,omitempty"`
	WithMigration string `json:"includes_migration,omitempty"`
	// Feature Flag Change Related Fields:
	FFChangeType string `json:"change_type,omitempty"`
	FeatureName  string `json:"feature_name,omitempty"`
	IsForAllOrgs string `json:"is_for_all_orgs,omitempty"`
	// Postback Update & Feature Flag Change Related Fields:
	Orgs           string `json:"orgs,omitempty"`
	AdditionalInfo string `json:"additional_info,omitempty"`
	Query          string `json:"query,omitempty"`
	// Postback Update Related Fields:
	PostbackUpdateType string `json:"postback_update_type,omitempty"`
	AdNetwork          string `json:"ad_net,omitempty"`
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

func ParseJSONToMessage(jsonText string) Message {
	jsonText = utils.ParseAndFixQueryInMessage(utils.ReplaceCommasWithQuotationMarks(jsonText))
	fmt.Println("Message (after replacing commas with quotation marks): ", jsonText) // for debugging

	var message Message
	err := json.Unmarshal([]byte(jsonText), &message)
	if err != nil {
		log.Printf("Error parsing JSON: %s", err)
	}
	return message
}

func (m *Message) ToDeploy() StructuredMessage {
	var deploy StructuredMessage
	deploy.ID = uuid.NewV4().String()
	deploy.Type = m.Type
	deploy.Subtype = "merge"
	deploy.Description = m.Description
	deploy.Timestamps = time.Now().Format(time.RFC3339)

	pr := utils.RemoveFirstCharAndLastChar(m.PullRequest)

	deploy.Subject = Subject{Type: "repo", Name: utils.GetRepoNameFromGithubPrURL(pr)}

	relatedSubjects := make([]RelatedSubject, 0)
	if m.PullRequest != "" {
		relatedSubject := RelatedSubject{Type: "pull_request", StringValue: pr}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	if m.Components != "" {
		relatedSubject := RelatedSubject{Type: "components", ArrayValue: strings.Split(m.Components, ",")}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}
	relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "has_migration", StringValue: strconv.FormatBool(utils.TranslateToBoolean(m.WithMigration))}) //TODO: fix this!
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
		{Type: "is_for_all_orgs", StringValue: strconv.FormatBool(utils.TranslateToBoolean(m.IsForAllOrgs))},
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

func (m *Message) ToPostbackUpdate() StructuredMessage {
	var postbackUpdate StructuredMessage
	postbackUpdate.ID = uuid.NewV4().String()
	postbackUpdate.Type = m.Type
	postbackUpdate.Subtype = m.PostbackUpdateType
	postbackUpdate.Timestamps = time.Now().Format(time.RFC3339)

	table_name := "postback_" + strings.ToLower(m.PostbackUpdateType)

	relatedSubjects := []RelatedSubject{
		{Type: "query", StringValue: strings.ReplaceAll(m.Query, "\n", " ")},
	}

	orgs_array := []string{}
	ad_net := m.AdNetwork
	subjectDescription := ""

	if m.Orgs != "" {
		orgs_array = strings.Split(m.Orgs, ",")
		relatedSubject := RelatedSubject{Type: "orgs", ArrayValue: orgs_array}
		relatedSubjects = append(relatedSubjects, relatedSubject)
	}

	if m.AdNetwork != "" {
		relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "ad_network", StringValue: m.AdNetwork})
	}

	if m.AdditionalInfo != "" {
		relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "additional_info", StringValue: m.AdditionalInfo})
	}

	if len(orgs_array) == 0 && ad_net == "" {
		postbackUpdate.Subject = Subject{Type: "table", Name: table_name}
	} else {
		relatedSubjects = append(relatedSubjects, RelatedSubject{Type: "table", StringValue: table_name})
		if len(orgs_array) == 0 {
			postbackUpdate.Subject = Subject{Type: "Ad Network", Name: ad_net}
			subjectDescription = " for Ad Network " + ad_net
		} else {
			if len(orgs_array) == 1 {
				postbackUpdate.Subject = Subject{Type: "org", Name: orgs_array[0]}
				subjectDescription = " for The Organization " + orgs_array[0]
			} else {
				postbackUpdate.Subject = Subject{Type: "orgs", Name: strings.Join(orgs_array, ", ")}
				subjectDescription = " for The Organizations " + strings.Join(orgs_array, ", ")
			}
		}
	}

	postbackUpdate.RelatedSubjects = relatedSubjects
	postbackUpdate.Description = "Updating Postback " + m.PostbackUpdateType + subjectDescription + "."

	return postbackUpdate
}

func (sm *StructuredMessage) Write_structured_json_to_file(filepath string) {
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
