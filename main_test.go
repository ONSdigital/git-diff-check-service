package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

const (
	testSecret = "secret"
)

func TestMain(m *testing.M) {
	// Fake credentials that we're expecting to run
	secret = []byte(testSecret)
	os.Exit(m.Run())
}

func TestPush(t *testing.T) {
	r := mux.NewRouter()
	setupRoutes(r)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Must use the url supplied by the test server or we will be unable to
	// parse the parameters from it.
	url := ts.URL + "/push"

	// TODO
	var jsonPayload = exampleHook

	// Sign the body so it'll be accepted
	signature := signPayload(jsonPayload, secret)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		t.Fatal("Failed to create http post:", err)
	}
	req.Header.Set("Content-Tyoe", "application/json")
	req.Header.Set("x-hub-signature", "sha1="+signature)
	req.Header.Set("x-github-event", "push")
	req.Header.Set("x-github-delivery", "667142b0-df17-11e7-9c7f-1ab48d642359") // fake

	client := &http.Client{
		Timeout: time.Second * 5,
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal("Failed to post to endpoint:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected status response, got %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) != "" {
		t.Errorf("unexpected response body, got '%s', expected '%s'", body, "")
	}
}

func TestUtilities(t *testing.T) {
	expected := "5d61605c3feea9799210ddcb71307d4ba264225f"
	payload := []byte("{}")
	if s := signPayload(payload, secret); s != expected {
		t.Errorf("Signature compute incorrect, got %s, expected %s", s, expected)
	}
}

// Taken from https://developer.github.com/v3/activity/events/types/#pushevent
var exampleHook = []byte(`
{
	"ref": "refs/heads/changes",
	"before": "9049f1265b7d61be4a8904a9a27120d2064dab3b",
	"after": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
	"created": false,
	"deleted": false,
	"forced": false,
	"base_ref": null,
	"compare": "https://github.com/baxterthehacker/public-repo/compare/9049f1265b7d...0d1a26e67d8f",
	"commits": [
	  {
		"id": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
		"tree_id": "f9d2a07e9488b91af2641b26b9407fe22a451433",
		"distinct": true,
		"message": "Update README.md",
		"timestamp": "2015-05-05T19:40:15-04:00",
		"url": "https://github.com/baxterthehacker/public-repo/commit/0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
		"author": {
		  "name": "baxterthehacker",
		  "email": "baxterthehacker@users.noreply.github.com",
		  "username": "baxterthehacker"
		},
		"committer": {
		  "name": "baxterthehacker",
		  "email": "baxterthehacker@users.noreply.github.com",
		  "username": "baxterthehacker"
		},
		"added": [
  
		],
		"removed": [
  
		],
		"modified": [
		  "README.md"
		]
	  }
	],
	"head_commit": {
	  "id": "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
	  "tree_id": "f9d2a07e9488b91af2641b26b9407fe22a451433",
	  "distinct": true,
	  "message": "Update README.md",
	  "timestamp": "2015-05-05T19:40:15-04:00",
	  "url": "https://github.com/baxterthehacker/public-repo/commit/0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c",
	  "author": {
		"name": "baxterthehacker",
		"email": "baxterthehacker@users.noreply.github.com",
		"username": "baxterthehacker"
	  },
	  "committer": {
		"name": "baxterthehacker",
		"email": "baxterthehacker@users.noreply.github.com",
		"username": "baxterthehacker"
	  },
	  "added": [
  
	  ],
	  "removed": [
  
	  ],
	  "modified": [
		"README.md"
	  ]
	},
	"repository": {
	  "id": 35129377,
	  "name": "public-repo",
	  "full_name": "baxterthehacker/public-repo",
	  "owner": {
		"name": "baxterthehacker",
		"email": "baxterthehacker@users.noreply.github.com"
	  },
	  "private": false,
	  "html_url": "https://github.com/baxterthehacker/public-repo",
	  "description": "",
	  "fork": false,
	  "url": "https://github.com/baxterthehacker/public-repo",
	  "forks_url": "https://api.github.com/repos/baxterthehacker/public-repo/forks",
	  "keys_url": "https://api.github.com/repos/baxterthehacker/public-repo/keys{/key_id}",
	  "collaborators_url": "https://api.github.com/repos/baxterthehacker/public-repo/collaborators{/collaborator}",
	  "teams_url": "https://api.github.com/repos/baxterthehacker/public-repo/teams",
	  "hooks_url": "https://api.github.com/repos/baxterthehacker/public-repo/hooks",
	  "issue_events_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues/events{/number}",
	  "events_url": "https://api.github.com/repos/baxterthehacker/public-repo/events",
	  "assignees_url": "https://api.github.com/repos/baxterthehacker/public-repo/assignees{/user}",
	  "branches_url": "https://api.github.com/repos/baxterthehacker/public-repo/branches{/branch}",
	  "tags_url": "https://api.github.com/repos/baxterthehacker/public-repo/tags",
	  "blobs_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/blobs{/sha}",
	  "git_tags_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/tags{/sha}",
	  "git_refs_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/refs{/sha}",
	  "trees_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/trees{/sha}",
	  "statuses_url": "https://api.github.com/repos/baxterthehacker/public-repo/statuses/{sha}",
	  "languages_url": "https://api.github.com/repos/baxterthehacker/public-repo/languages",
	  "stargazers_url": "https://api.github.com/repos/baxterthehacker/public-repo/stargazers",
	  "contributors_url": "https://api.github.com/repos/baxterthehacker/public-repo/contributors",
	  "subscribers_url": "https://api.github.com/repos/baxterthehacker/public-repo/subscribers",
	  "subscription_url": "https://api.github.com/repos/baxterthehacker/public-repo/subscription",
	  "commits_url": "https://api.github.com/repos/baxterthehacker/public-repo/commits{/sha}",
	  "git_commits_url": "https://api.github.com/repos/baxterthehacker/public-repo/git/commits{/sha}",
	  "comments_url": "https://api.github.com/repos/baxterthehacker/public-repo/comments{/number}",
	  "issue_comment_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues/comments{/number}",
	  "contents_url": "https://api.github.com/repos/baxterthehacker/public-repo/contents/{+path}",
	  "compare_url": "https://api.github.com/repos/baxterthehacker/public-repo/compare/{base}...{head}",
	  "merges_url": "https://api.github.com/repos/baxterthehacker/public-repo/merges",
	  "archive_url": "https://api.github.com/repos/baxterthehacker/public-repo/{archive_format}{/ref}",
	  "downloads_url": "https://api.github.com/repos/baxterthehacker/public-repo/downloads",
	  "issues_url": "https://api.github.com/repos/baxterthehacker/public-repo/issues{/number}",
	  "pulls_url": "https://api.github.com/repos/baxterthehacker/public-repo/pulls{/number}",
	  "milestones_url": "https://api.github.com/repos/baxterthehacker/public-repo/milestones{/number}",
	  "notifications_url": "https://api.github.com/repos/baxterthehacker/public-repo/notifications{?since,all,participating}",
	  "labels_url": "https://api.github.com/repos/baxterthehacker/public-repo/labels{/name}",
	  "releases_url": "https://api.github.com/repos/baxterthehacker/public-repo/releases{/id}",
	  "created_at": 1430869212,
	  "updated_at": "2015-05-05T23:40:12Z",
	  "pushed_at": 1430869217,
	  "git_url": "git://github.com/baxterthehacker/public-repo.git",
	  "ssh_url": "git@github.com:baxterthehacker/public-repo.git",
	  "clone_url": "https://github.com/baxterthehacker/public-repo.git",
	  "svn_url": "https://github.com/baxterthehacker/public-repo",
	  "homepage": null,
	  "size": 0,
	  "stargazers_count": 0,
	  "watchers_count": 0,
	  "language": null,
	  "has_issues": true,
	  "has_downloads": true,
	  "has_wiki": true,
	  "has_pages": true,
	  "forks_count": 0,
	  "mirror_url": null,
	  "open_issues_count": 0,
	  "forks": 0,
	  "open_issues": 0,
	  "watchers": 0,
	  "default_branch": "master",
	  "stargazers": 0,
	  "master_branch": "master"
	},
	"pusher": {
	  "name": "baxterthehacker",
	  "email": "baxterthehacker@users.noreply.github.com"
	},
	"sender": {
	  "login": "baxterthehacker",
	  "id": 6752317,
	  "avatar_url": "https://avatars.githubusercontent.com/u/6752317?v=3",
	  "gravatar_id": "",
	  "url": "https://api.github.com/users/baxterthehacker",
	  "html_url": "https://github.com/baxterthehacker",
	  "followers_url": "https://api.github.com/users/baxterthehacker/followers",
	  "following_url": "https://api.github.com/users/baxterthehacker/following{/other_user}",
	  "gists_url": "https://api.github.com/users/baxterthehacker/gists{/gist_id}",
	  "starred_url": "https://api.github.com/users/baxterthehacker/starred{/owner}{/repo}",
	  "subscriptions_url": "https://api.github.com/users/baxterthehacker/subscriptions",
	  "organizations_url": "https://api.github.com/users/baxterthehacker/orgs",
	  "repos_url": "https://api.github.com/users/baxterthehacker/repos",
	  "events_url": "https://api.github.com/users/baxterthehacker/events{/privacy}",
	  "received_events_url": "https://api.github.com/users/baxterthehacker/received_events",
	  "type": "User",
	  "site_admin": false
	}
  }
`)
