package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"time"
)

const N_RECENT_SUBMISSIONS = 15

func Handler(w http.ResponseWriter, r *http.Request) {
	basicAuth(Integrator)
}

func basicAuth(next func()) http.HandlerFunc {
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
				next()
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func Integrator() {
	nextRepitition := map[string]string{"1": "7", "7": "30", "30": "90", "90": "180", "180": "365", "365": "Done"}

	notionHeaders := http.Header{
		"Accept":         {"application/json"},
		"Notion-Version": {"2022-02-22"},
		"Content-Type":   {"application/json"},
		"Authorization":  {"Bearer " + os.Getenv("PERSONAL_NOTION_TOKEN")},
	}

	recentSubmissions := getRecentSubmissions(os.Getenv("LC_USERNAME"), N_RECENT_SUBMISSIONS)

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

func timestampToFormat(stamp string, format string) string {
	timestamp, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil {
		panic(err)
	}
	tm := time.Unix(timestamp, 0)
	return tm.Format(format)
}
