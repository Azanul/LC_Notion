package lcnotion

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"time"
)

const LC_URL = "https://leetcode.com/graphql/"
const NOTION_URL = "https://api.notion.com/v1"
const N_RECENT_SUBMISSIONS = 15

func Handler(w http.ResponseWriter, r *http.Request) {
	basicAuth(Integrator)
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

func Integrator(w http.ResponseWriter, r *http.Request) {
	nextRepitition := map[string]string{"1": "7", "7": "30", "30": "90", "90": "180", "180": "365", "365": "Done"}

	notionToken := os.Getenv("personal_notion_token")
	dbId := os.Getenv("personal_db_id")
	lcUsername := os.Getenv("lc_username")

	notionHeaders := map[string]string{
		"Accept":         "application/json",
		"Notion-Version": "2022-02-22",
		"Content-Type":   "application/json",
		"Authorization":  "Bearer" + notion_token,
	}

	client := http.Client{
		Timeout: 60 * time.Second,
	}

	recentSubmissions := getRecentSubmissions(client)

	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		os.Exit(1)
	}
}

func getRecentSubmissions(sender http.Client) {
    query := `
		query recentAcSubmissions($username: String!, $limit: Int!) {
            recentAcSubmissionList(username: $username, limit: $limit) {
                titleSlug
                timestamp
            }
        }
    `

    // variables := map[string]string{ "username": lc_username, "limit": lc_recent_subs_limit }

    // r = requests.post(LC_URL, json={"query": query, "variables": variables})

	jsonBody := []byte(`{"query": query, "variables": variables}`)
	bodyReader := bytes.NewReader(jsonBody)

	req, err := http.NewRequest(http.MethodPost, LC_URL, bodyReader)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
    return json.loads(r.text)['data']['recentAcSubmissionList']
}