package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Azanul/lcnotion/api/internal"
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

	recentSubmissions := internal.GetRecentSubmissions(os.Getenv("LC_USERNAME"), N_RECENT_SUBMISSIONS)

	titleSlugs := make(map[string]string)
	for i := len(recentSubmissions) - 1; i >= 0; i-- {
		titleSlugs[recentSubmissions[i]["titleSlug"]] = recentSubmissions[i]["timestamp"]
	}

	slugFilters := []internal.SlugFilter{}
	for ts := range titleSlugs {
		slugFilters = append(slugFilters,
			internal.SlugFilter{
				Property: "titleSlug",
				RichText: map[string]string{
					"equals": ts,
				},
			},
		)
	}

	matchingEntries := internal.GetEntriesByFilter(slugFilters, notionHeaders, N_RECENT_SUBMISSIONS)

	for _, uq := range matchingEntries.Results {
		reviewDate := timestampToFormat(titleSlugs[uq.Properties.TitleSlug.RichText[0].PlainText], "2006-01-02")

		if reviewDate != uq.Properties.LastReviewed.Date["start"] {
			updateProperties := map[string]internal.QuestionProperties{
				"properties": {
					LastReviewed: &internal.DateField{
						Date: map[string]string{
							"start": reviewDate,
						},
					},
					RepetitionGap: &internal.SelectField{
						Select: map[string]string{
							"name": nextRepitition[uq.Properties.RepetitionGap.Select["name"]],
						},
					},
				},
			}

			internal.UpdateExistingEntry(uq.Id, updateProperties, notionHeaders)
		}

		delete(titleSlugs, uq.Properties.TitleSlug.RichText[0].PlainText)
	}

	for slug, timestamp := range titleSlugs {
		quesJson := internal.GetQuestionBySlug(slug)

		newProperties := internal.QuestionProperties{
			TitleSlug: &internal.TextField{
				RichText: []internal.RichText{
					{
						Type: "text",
						Text: map[string]string{
							"content": slug,
						},
						Annotations: internal.Annotation{
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
			LastReviewed: &internal.DateField{
				Date: map[string]string{
					"start": timestampToFormat(timestamp, "2006-01-02"),
				},
			},
			RepetitionGap: &internal.SelectField{
				Select: map[string]string{
					"name": "1",
				},
			},
			Level: &internal.SelectField{
				Select: map[string]string{
					"name": quesJson.Difficulty,
				},
			},
			Source: &internal.SelectField{
				Select: map[string]string{
					"name": "Website",
				},
			},
			Materials: &internal.MediaField{
				Files: []internal.FileField{
					{
						Name: "https://leetcode.com/problems/" + slug + "/",
						Type: "external",
						External: map[string]string{
							"url": "https://leetcode.com/problems/" + slug + "/",
						},
					},
				},
			},
			Name: &internal.TitleField{
				Title: []internal.RichText{
					{
						Text: map[string]string{
							"content": quesJson.QuestionId + ". " + quesJson.Title,
						},
					},
				},
			},
		}

		internal.CreateNewEntry(newProperties, notionHeaders)
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
