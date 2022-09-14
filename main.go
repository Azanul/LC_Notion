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
	Property string
	RichText map[string]string
}

type filter struct {
	PageSize int
	Filter   map[string][]slugFilter
}

type questionProperties struct {
}

type question struct {
	Id,
	Properties questionProperties
}

func Integrator(w http.ResponseWriter, r *http.Request) {
	// nextRepitition := map[string]string{"1": "7", "7": "30", "30": "90", "90": "180", "180": "365", "365": "Done"}

	recentSubmissions := getRecentSubmissions(os.Getenv("LC_USERNAME"))

	titleSlugs := make(map[string]string)
	for i := len(recentSubmissions) - 1; i >= 0; i-- {
		titleSlugs[recentSubmissions[i]["titleSlug"]] = recentSubmissions[i]["timestamp"]
	}

	slugFilters := []slugFilter{}
	for _, ts := range titleSlugs {
		slugFilters = append(slugFilters,
			slugFilter{
				Property: "titleSlug",
				RichText: map[string]string{
					"equals": ts,
				},
			},
		)
	}

	payload := filter{
		PageSize: 20,
		Filter:   map[string][]slugFilter{"or": slugFilters},
	}

	postBody, _ := json.Marshal(payload)
	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(http.MethodPost, NOTION_URL+"/databases"+os.Getenv("PERSONAL_DB_ID")+"query", requestBody)
	if err != nil {
		fmt.Printf("client: could not send request: %s\n", err)
		os.Exit(1)
	}
	req.Header = http.Header{
		"Accept":         {"application/json"},
		"Notion-Version": {"2022-02-22"},
		"Content-Type":   {"application/json"},
		"Authorization":  {"Bearer" + os.Getenv("PERSONAL_NOTION_TOKEN")},
	}
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
	jsonBody := make(map[string]map[string][]map[string]string)
	_ = json.Unmarshal(body, &jsonBody)
	fmt.Println(jsonBody)
	// r = requests.post(f"{notion_url}/databases/{db_id}/query", json=payload, headers=notion_headers)

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
