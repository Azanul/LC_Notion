package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const LC_URL = "https://leetcode.com/graphql/"
const NOTION_URL = "https://api.notion.com/v1"
const N_RECENT_SUBMISSIONS = 15

func main() {
	godotenv.Load("test.env")
	http.HandleFunc("/", basicAuth(Integrator))

	err := http.ListenAndServe(":3333", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(os.Getenv("AUTH_USERNAME")))
			expectedPasswordHash := sha256.Sum256([]byte(os.Getenv("AUTH_PASSWORD")))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

type slugFilter struct {
	Property string            `json:"property"`
	RichText map[string]string `json:"rich_text"`
}

type filter struct {
	PageSize int                     `json:"page_size"`
	Filter   map[string][]slugFilter `json:"filter"`
}

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

type question struct {
	Difficulty       string              `json:"difficulty,omitempty"`
	QuestionId       string              `json:"questionId,omitempty"`
	SimilarQuestions string              `json:"similarQuestions,omitempty"`
	Title            string              `json:"title,omitempty"`
	TopicTags        []map[string]string `json:"topicTags,omitempty"`
	TitleSlug        string              `json:"titleSlug,omitempty"`
}

func Integrator(w http.ResponseWriter, r *http.Request) {
	nextRepitition := map[string]string{"1": "7", "7": "30", "30": "90", "90": "180", "180": "365", "365": "Done"}

	notionHeaders := http.Header{
		"Accept":         {"application/json"},
		"Notion-Version": {"2022-02-22"},
		"Content-Type":   {"application/json"},
		"Authorization":  {"Bearer " + os.Getenv("PERSONAL_NOTION_TOKEN")},
	}

	recentSubmissions := getRecentSubmissions(os.Getenv("LC_USERNAME"))

	titleSlugs := make(map[string]string)
	for i := len(recentSubmissions) - 1; i >= 0; i-- {
		titleSlugs[recentSubmissions[i]["titleSlug"]] = recentSubmissions[i]["timestamp"]
	}

	slugFilters := []slugFilter{}
	for ts := range titleSlugs {
		slugFilters = append(slugFilters,
			slugFilter{
				Property: "titleSlug",
				RichText: map[string]string{
					"equals": ts,
				},
			},
		)
	}

	matchingEntries := getEntriesBySlug(slugFilters, notionHeaders, N_RECENT_SUBMISSIONS)

	for _, uq := range matchingEntries.Results {
		reviewDate := timestampToFormat(titleSlugs[uq.Properties.TitleSlug.RichText[0].PlainText], "2006-01-02")

		if reviewDate != uq.Properties.LastReviewed.Date["start"] {
			delete(uq.Properties.TitleSlug.RichText[0].Text, "link")
			updateProperties := map[string]questionProperties{
				"properties": {
					LastReviewed: &dateField{
						Date: map[string]string{
							"start": reviewDate,
						},
					},
					RepetitionGap: &selectField{
						Select: map[string]string{
							"name": nextRepitition[uq.Properties.RepetitionGap.Select["name"]],
						},
					},
				},
			}

			updateExistingEntry(uq.Id, updateProperties, notionHeaders)
		}

		delete(titleSlugs, uq.Properties.TitleSlug.RichText[0].PlainText)
	}

	for slug, timestamp := range titleSlugs {
		quesJson := getQuestionBySlug(slug)

		newProperties := questionProperties{
			TitleSlug: &textField{
				RichText: []richText{
					{
						Type: "text",
						Text: map[string]string{
							"content": slug,
						},
						Annotations: annotation{
							Bold:          false,
							Italic:        false,
							Strikethrough: false,
							Underline:     false,
							Code:          false,
							Color:         "default",
						},
						PlainText: slug,
					},
				},
			},
			LastReviewed: &dateField{
				Date: map[string]string{
					"start": timestampToFormat(timestamp, "2006-01-02"),
				},
			},
			RepetitionGap: &selectField{
				Select: map[string]string{
					"name": "1",
				},
			},
			Level: &selectField{
				Select: map[string]string{
					"name": quesJson.Difficulty,
				},
			},
			Source: &selectField{
				Select: map[string]string{
					"name": "Website",
				},
			},
			Materials: &mediaField{
				Files: []fileField{
					{
						Name: "https://leetcode.com/problems/" + slug + "/",
						Type: "external",
						External: map[string]string{
							"url": "https://leetcode.com/problems/" + slug + "/",
						},
					},
				},
			},
			Name: &titleField{
				Title: []richText{
					{
						Text: map[string]string{
							"content": quesJson.QuestionId + ". " + quesJson.Title,
						},
					},
				},
			},
		}

		createNewEntry(newProperties, notionHeaders)
	}
}

func getRecentSubmissions(lcUsername string) []map[string]string {
	query := `
		query recentAcSubmissions($username: String!, $limit: Int!) {
            recentAcSubmissionList(username: $username, limit: $limit) {
                titleSlug
                timestamp
            }
        }
    `

	variables := fmt.Sprintf(`{"username": "%s", "limit": %d}`, lcUsername, N_RECENT_SUBMISSIONS)
	postBody, _ := json.Marshal(map[string]string{
		"query":     query,
		"variables": variables,
	})
	requestBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(LC_URL, "application/json", requestBody)
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
	jsonBody := make(map[string]map[string][]map[string]string)
	_ = json.Unmarshal(body, &jsonBody)
	return jsonBody["data"]["recentAcSubmissionList"]
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

func getQuestionBySlug(titleSlug string) question {
	query := `
    query questionData($titleSlug: String!) {
        question(titleSlug: $titleSlug) {
            questionId
            title
            difficulty
            similarQuestions
            topicTags {
                name
            }
        }
    }
    `

	variables := fmt.Sprintf(`{"titleSlug": "%s"}`, titleSlug)
	postBody, _ := json.Marshal(map[string]string{
		"query":     query,
		"variables": variables,
	})
	requestBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(LC_URL, "application/json", requestBody)
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
	jsonBody := make(map[string]map[string]question)
	_ = json.Unmarshal(body, &jsonBody)
	return jsonBody["data"]["question"]
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

func timestampToFormat(stamp string, format string) string {
	timestamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		panic(err)
	}
	tm := time.Unix(timestamp, 0)
	return tm.Format(format)
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
