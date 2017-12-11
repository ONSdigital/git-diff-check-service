package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/ONSdigital/git-diff-check-service/api"
	"github.com/ONSdigital/git-diff-check-service/signals"
	"github.com/ONSdigital/git-diff-check/diffcheck"

	githubhook "gopkg.in/rjz/githubhook.v0"

	"github.com/gorilla/mux"
)

var (
	secret []byte // Webhook secret
)

const (
	githubURL = "https://api.github.com"
)

type (
	// GithubUser represents a github user. This will be embedded in other
	// returned data. However, not all fields will always be filled in.
	GithubUser struct {
		Name      string `json:"name,omitempty"`
		Email     string `json:"email,omitempty"`
		Login     string `json:"login,omitempty"`
		ID        int    `json:"id,omitempty"`
		AvatarURL string `json:"avatar_url,omitempty"`
	}

	// PushEvent is the data received from a github "push" webhook
	PushEvent struct {
		Ref     string `json:"id"`
		Commits []struct {
			ID string `json:"id"`
		} `json:"commits"`
		Repository struct {
			ID       int        `json:"id"`
			Name     string     `json:"name"`
			FullName string     `json:"full_name"`
			Owner    GithubUser `json:"owner"`
		}
		Pusher GithubUser `json:"pusher"`
	}

	// Commit is a single repository commit
	Commit struct {
		URL   string `json:"url"`
		SHA   string `json:"sha"`
		Files []struct {
			Filename  string `json:"filename"`
			Additions int    `json:"additions"`
			Deletions int    `json:"deletions"`
			Changes   int    `json:"changes"`
			Status    string `json:"status"`
			Patch     string `json:"patch"`
		} `json:"files"`
		Commit struct {
			Author    GithubUser `json:"author"`
			Committer GithubUser `json:"committer"`
		} `json:"commit"`
		Message string `json:"message"`
	}
)

func main() {

	// Signals
	// Set up the signal handler to watch for SIGTERM and SIGINT signals so we
	// can at least attempt to gracefully shut down before the PaaS/docker etc
	// running us unceremoneously kills us with a SIGKILL.
	cancelSigWatch := signals.HandleFunc(
		func(sig os.Signal) {
			log.Printf(`event="Shutting down" signal="%s"`, sig.String())

			// TODO - Any necessary clean up or waiting on outstanding goroutines
			//		  May need to fire off some context cancels here

			log.Print(`event="Exiting"`)
			os.Exit(0)
		},
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancelSigWatch()

	// Import env
	var port string
	if port = os.Getenv("PORT"); len(port) == 0 {
		log.Fatal(`event="Failed to start" error="missing PORT env"`)
	}

	if secret = []byte(os.Getenv("WEBHOOK_SECRET")); len(secret) == 0 {
		log.Fatal(`event="Failed to start" error="missing WEBHOOK_SECRET env"`)
	}

	// Webserver
	r := mux.NewRouter()
	r.HandleFunc("/push", pushHandler)
	http.Handle("/", r)
	log.Printf(`event="Started" port="%s"`, port)
	log.Fatalf(`event="Stopped" error="%v"`, http.ListenAndServe(":"+port, nil))
}

func pushHandler(rw http.ResponseWriter, r *http.Request) {
	log.Println(`event="Received push event"`)

	hook, err := githubhook.Parse(secret, r)
	if err != nil {
		log.Printf(`event="Error parsing hook" error="%v"`, err)
		api.WriteProblemResponse(api.Problem{
			Title:  "Request body unreadable",
			Status: http.StatusInternalServerError,
		}, rw)
		return
	}

	var pushEvent PushEvent
	if err := json.Unmarshal(hook.Payload, &pushEvent); err != nil {
		log.Printf(`event="Error parsing hook payload" error="%v"`, err)
		api.WriteProblemResponse(api.Problem{
			Title:  "Failed to unmarshal payload",
			Status: http.StatusInternalServerError,
		}, rw)
		return
	}

	for _, commit := range pushEvent.Commits {
		// Warning: Be careful if you try to run this under a lambda type (e.g.
		// deploying with apex) as these go routines will probably be killed as
		// soon as the parent function returns.
		go checkCommit(pushEvent.Repository.FullName, commit.ID)
	}

	rw.WriteHeader(http.StatusOK)
	return
}

func checkCommit(repo, sha string) {

	bindLog := fmt.Sprintf(`repo="%s" sha="%s"`, repo, sha)

	log.Printf(`event="Checking commit" %s`, bindLog)

	client := &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(githubURL + "/repos/" + repo + "/commits/" + sha)
	if err != nil {
		log.Printf(`event="Failed to retrieve commit info" %s error="%v"`, bindLog, err)
		// TODO need to determine what to do if this fails
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf(`event="Failed to retrieve commit info" %s error="%v"`, bindLog, err)
		// TODO need to determine what to do if this fails
		return
	}

	var c Commit
	if err := json.Unmarshal(body, &c); err != nil {
		log.Printf(`event="Failed to parse commit info" %s error="%v"`, bindLog, err)
		// TODO need to determine what to do if this fails
		return
	}

	ok, reports, err := diffcheck.SnoopPatch([]byte(c.Files[0].Patch))
	if err != nil {
		log.Printf(`event="Failed to snoop commit info" %s error="%v"`, bindLog, err)
		// TODO need to determine what to do if this fails
		return
	}

	log.Printf(`event="Snoop complete" sha="%s" ok="%v"`, sha, ok)
	if !ok {
		for _, report := range reports {
			for _, warning := range report.Warnings {
				log.Printf(`event="Warning found" %s warning="%s" type="%s" line="%d"`, bindLog, warning.Description, warning.Type, warning.Line)
			}
		}
		return
	}

	log.Printf(`event="No issues found in commit" %s`, bindLog)
	return

	// TODO - Need to actually do something sensible with the reports - like alert someone!
	// TODO - Though splunk et al could be used in the short term to alert on logged above
}
