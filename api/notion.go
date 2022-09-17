package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const NOTION_URL = "https://api.notion.com/v1"

type annotation struct {
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Color         string `json:"color,omitempty"`
}

type richText struct {
	Type        string            `json:"type,omitempty"`
	Text        map[string]string `json:"text,omitempty"`
	Annotations annotation        `json:"annotations,omitempty"`
	PlainText   string            `json:"plain_text,omitempty"`
	Href        string            `json:"href,omitempty"`
}

type textField struct {
	Id       string     `json:"id,omitempty"`
	Type     string     `json:"type,omitempty"`
	RichText []richText `json:"rich_text,omitempty"`
}

type titleField struct {
	Id    string     `json:"id,omitempty"`
	Type  string     `json:"type,omitempty"`
	Title []richText `json:"title,omitempty"`
}

type selectField struct {
	Id     string            `json:"id,omitempty"`
	Type   string            `json:"type,omitempty"`
	Select map[string]string `json:"select,omitempty"`
}

type dateField struct {
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

type fileField struct {
	Name     string            `json:"name,omitempty"`
	Type     string            `json:"type,omitempty"`
	External map[string]string `json:"external,omitempty"`
}

type mediaField struct {
	Id    string      `json:"id,omitempty"`
	Type  string      `json:"type,omitempty"`
	Files []fileField `json:"files,omitempty"`
}

type questionProperties struct {
	TitleSlug           *textField         `json:"titleSlug,omitempty"`
	RepetitionGap       *selectField       `json:"Repetition Gap,omitempty"`
	Level               *selectField       `json:"Level,omitempty"`
	DaysSinceLastReview *formulaField      `json:"Days Since last review,omitempty"`
	Created             *map[string]string `json:"Created,omitempty"`
	Source              *selectField       `json:"Source,omitempty"`
	LastReviewed        *dateField         `json:"Last Reviewed,omitempty"`
	Materials           *mediaField        `json:"Materials,omitempty"`
	Name                *titleField        `json:"Name,omitempty"`
}

type questionEntry struct {
	Id         string             `json:"id"`
	Properties questionProperties `json:"properties"`
}

type notionResponse struct {
	Object  string          `json:"object"`
	Results []questionEntry `json:"results"`
}

type slugFilter struct {
	Property string            `json:"property"`
	RichText map[string]string `json:"rich_text"`
}

type filter struct {
	PageSize int                     `json:"page_size"`
	Filter   map[string][]slugFilter `json:"filter"`
}

func createNewEntry(properties questionProperties, header http.Header) {
	payload := struct {
		Parent     map[string]string  `json:"parent"`
		Properties questionProperties `json:"properties"`
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

func getEntriesBySlug(slugFilters []slugFilter, header http.Header, pageSize int) notionResponse {
	payload := filter{
		PageSize: N_RECENT_SUBMISSIONS,
		Filter:   map[string][]slugFilter{"or": slugFilters},
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

func updateExistingEntry(_id string, properties map[string]questionProperties, header http.Header) {
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
