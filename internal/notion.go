package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const NOTION_URL = "https://api.notion.com/v1"

type Annotation struct {
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Color         string `json:"color,omitempty"`
}

type RichText struct {
	Type        string            `json:"type,omitempty"`
	Text        map[string]string `json:"text,omitempty"`
	Annotations Annotation        `json:"annotations,omitempty"`
	PlainText   string            `json:"plain_text,omitempty"`
	Href        string            `json:"href,omitempty"`
}

type TextField struct {
	Id       string     `json:"id,omitempty"`
	Type     string     `json:"type,omitempty"`
	RichText []RichText `json:"rich_text,omitempty"`
}

type TitleField struct {
	Id    string     `json:"id,omitempty"`
	Type  string     `json:"type,omitempty"`
	Title []RichText `json:"title,omitempty"`
}

type SelectField struct {
	Id     string            `json:"id,omitempty"`
	Type   string            `json:"type,omitempty"`
	Select map[string]string `json:"select,omitempty"`
}

type DateField struct {
	Id   string            `json:"id,omitempty"`
	Type string            `json:"type,omitempty"`
	Date map[string]string `json:"date,omitempty"`
}

type formula struct {
	Type   string `json:"type,omitempty"`
	Number int    `json:"number,omitempty"`
}

type formulaField struct {
	Id      string  `json:"id,omitempty"`
	Type    string  `json:"type,omitempty"`
	Formula formula `json:"formula,omitempty"`
}

type FileField struct {
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	External map[string]string `json:"external,omitempty"`
}

type MediaField struct {
	Id    string      `json:"id,omitempty"`
	Type  string      `json:"type,omitempty"`
	Files []FileField `json:"files,omitempty"`
}

type QuestionProperties struct {
	TitleSlug           *TextField         `json:"titleSlug,omitempty"`
	RepetitionGap       *SelectField       `json:"Repetition Gap,omitempty"`
	Level               *SelectField       `json:"Level,omitempty"`
	DaysSinceLastReview *formulaField      `json:"Days Since last review,omitempty"`
	Created             *map[string]string `json:"Created,omitempty"`
	Source              *SelectField       `json:"Source,omitempty"`
	LastReviewed        *DateField         `json:"Last Reviewed,omitempty"`
	Materials           *MediaField        `json:"Materials,omitempty"`
	Name                *TitleField        `json:"Name,omitempty"`
}

type questionEntry struct {
	Id         string             `json:"id"`
	Properties QuestionProperties `json:"properties"`
}

type notionResponse struct {
	Object  string          `json:"object"`
	Results []questionEntry `json:"results"`
}

type SlugFilter struct {
	Property string            `json:"property"`
	RichText map[string]string `json:"rich_text"`
}

type filter struct {
	PageSize int                     `json:"page_size"`
	Filter   map[string][]SlugFilter `json:"filter"`
}

func CreateNewEntry(properties QuestionProperties, header http.Header) {
	payload := struct {
		Parent     map[string]string  `json:"parent"`
		Properties QuestionProperties `json:"properties"`
	}{
		Parent: map[string]string{
			"database_id": os.Getenv("PERSONAL_DB_ID"),
		},
		Properties: properties,
	}
	fmt.Println(payload)
	postBody, _ := json.Marshal(payload)
	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPost, NOTION_URL+"/pages/", requestBody)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
}

func GetEntriesByFilter(slugFilters []SlugFilter, header http.Header, pageSize int) notionResponse {
	payload := filter{
		PageSize: pageSize,
		Filter:   map[string][]SlugFilter{"or": slugFilters},
	}

	postBody, _ := json.Marshal(payload)
	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPost, NOTION_URL+"/databases/"+os.Getenv("PERSONAL_DB_ID")+"/query", requestBody)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("client: could not read response: %s\n", err)
		os.Exit(1)
	}
	toBeUpdated := notionResponse{}
	_ = json.Unmarshal(body, &toBeUpdated)
	return toBeUpdated
}

func UpdateExistingEntry(_id string, properties map[string]QuestionProperties, header http.Header) {
	postBody, _ := json.Marshal(properties)
	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPatch, NOTION_URL+"/pages/"+_id, requestBody)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	req.Header = header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
}
