import os
import requests
import json
from datetime import datetime
from dotenv import load_dotenv, find_dotenv

load_dotenv(find_dotenv('.test.env'))

next_repitition = {"1": "7", "7": "30", "30": "90", "90": "180", "180": "365", "365": "Done"}
lc_url = 'https://leetcode.com/graphql/'
notion_url = "https://api.notion.com/v1"

lc_username = os.getenv('lc_username')
lc_recent_subs_limit = 15
notion_token = os.getenv('personal_notion_token')
db_id = os.getenv('personal_db_id')

notion_headers = {
        "Accept": "application/json",
        "Notion-Version": "2022-02-22",
        "Content-Type": "application/json",
        "Authorization": f"Bearer {notion_token}"
    }

def get_recent_subs():
    query = """
        query recentAcSubmissions($username: String!, $limit: Int!) {
            recentAcSubmissionList(username: $username, limit: $limit) {
                titleSlug
                timestamp
            }
        }
    """

    variables = { "username": lc_username, "limit": lc_recent_subs_limit }

    r = requests.post(lc_url, json={"query": query, "variables": variables})
    return json.loads(r.text)['data']['recentAcSubmissionList']


def get_question(title_slug):
    query = """
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
    """
    r = requests.post(lc_url, json={"query": query, "variables": {"titleSlug": title_slug}})
    return json.loads(r.text)['data']['question']

def patch_entries(_id, properties):
    r = requests.patch(f"{notion_url}/pages/{_id}", json={"properties": properties}, headers=notion_headers)

def post_entries(properties):
    r = requests.post(f"{notion_url}/pages/",
     json={"parent": { "database_id": db_id }, "properties": properties},
     headers=notion_headers)

def put_entries():
    recent_subs = get_recent_subs()
    title_slugs = {rs['titleSlug']: datetime.fromtimestamp(int(rs['timestamp'])) for rs in recent_subs[::-1]}
    
    slug_filter = []

    for ts in title_slugs:
        slug_filter.append({
            "property": "titleSlug",
            "rich_text": {
                "equals": ts
            }
        })

    payload = {
        "page_size": 20, 
        "filter": {"or": slug_filter}}

    

    r = requests.post(f"{notion_url}/databases/{db_id}/query", json=payload, headers=notion_headers)
    update_ques = json.loads(r.text)['results']
    
    for uq in update_ques:
        new_review_date = title_slugs[uq['properties']['titleSlug']['rich_text'][0]['plain_text']].strftime('%Y-%m-%d')
        
        if new_review_date != uq['properties']['Last Reviewed']['date']['start']:
            update_properties = {
                "Last Reviewed": {
                    "date": {
                        "start": new_review_date,
                        "end": None
                    }
                },
                "Repetition Gap": {
                    "select": {
                        "name": next_repitition[uq['properties']['Repetition Gap']['select']['name']]
                    }
                }
            }

            patch_entries(uq['id'], update_properties)

        del title_slugs[uq['properties']['titleSlug']['rich_text'][0]['plain_text']]
    
    for ts in title_slugs:
        ques_json = get_question(ts)

        new_properties = {
            "titleSlug": {
                "rich_text": [
                    {
                        "type": "text",
                        "text": {
                            "content": ts,
                            "link": None
                        },
                        "annotations": {
                            "bold": False,
                            "italic": False,
                            "strikethrough": False,
                            "underline": False,
                            "code": False,
                            "color": "default"
                        },
                        "plain_text": ts,
                        "href": None
                    }
                ]
            },
            "Last Reviewed": {
                "date": {
                    "start": title_slugs[ts].strftime('%Y-%m-%d'),
                    "end": None
                }
            },
            "Repetition Gap": {
                "select": {
                    "name": "1"
                }
            },
            "Level": {
                "select": {
                    "name": ques_json['difficulty']
                }
            },
            "Source": {
                "select": {
                    "name": "Website"
                }
            },
            "Materials": {
                "files": [
                    {
                        "name": f"https://leetcode.com/problems/{ts}/",
                        "type": "external",
                        "external": {
                            "url": f"https://leetcode.com/problems/{ts}/"
                        }
                    }
                ]
            },
            "Name": {
                "title": [
                    {
                        "text": {
                            "content": f"{ques_json['questionId']}. {ques_json['title']}"
                        }
                    }
                ]
            }
        }

        post_entries(new_properties)

put_entries()
