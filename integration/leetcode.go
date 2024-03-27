package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const LC_URL = "https://leetcode.com/graphql/"

type Question struct {
	Difficulty       string              `json:"difficulty,omitempty"`
	QuestionId       string              `json:"questionId,omitempty"`
	SimilarQuestions string              `json:"similarQuestions,omitempty"`
	Title            string              `json:"title,omitempty"`
	TopicTags        []map[string]string `json:"topicTags,omitempty"`
	TitleSlug        string              `json:"titleSlug,omitempty"`
}

func GetRecentSubmissions(lcUsername string, limit int) []map[string]string {
	query := `
		query recentAcSubmissions($username: String!, $limit: Int!) {
            recentAcSubmissionList(username: $username, limit: $limit) {
                titleSlug
                timestamp
            }
        }
    `

	variables := fmt.Sprintf(`{"username": "%s", "limit": %d}`, lcUsername, limit)
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

func GetQuestionBySlug(titleSlug string) Question {
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
	jsonBody := make(map[string]map[string]Question)
	_ = json.Unmarshal(body, &jsonBody)
	return jsonBody["data"]["question"]
}
