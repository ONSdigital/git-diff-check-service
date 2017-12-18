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
	"github.com/ONSdigital/git-diff-check-service/githook"
	"github.com/ONSdigital/git-diff-check-service/signals"
	"github.com/ONSdigital/git-diff-check/diffcheck"

	"github.com/gorilla/mux"
)

var (
	// Webhook secret - configured from envrionment and must match the secret
	// configured in github for the webhook.
	secret []byte
)

const (
	githubURL = "https://api.github.com"
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
	setupRoutes(r)
	http.Handle("/", r)
	log.Printf(`event="Started" port="%s"`, port)
	log.Fatalf(`event="Stopped" error="%v"`, http.ListenAndServe(":"+port, nil))
}

func setupRoutes(r *mux.Router) {
	r.HandleFunc("/push", pushHandler)
	r.HandleFunc("/healthcheck", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"healthy":true}`))
		rw.WriteHeader(http.StatusOK)
		return
	})
}

func pushHandler(rw http.ResponseWriter, r *http.Request) {
	event, err := githook.Parse(r, secret)
	if err != nil {
		log.Printf(`event="Error parsing hook" error="%v"`, err)
		api.WriteProblemResponse(api.Problem{
			Title:  "Problem parsing request body",
			Status: http.StatusBadRequest,
			Detail: err.Error(),
		}, rw)
		return
	}

	switch e := event.(type) {
	case *githook.PingEvent:
		log.Printf(`event="Received hook event" type="ping" repo="%s"`, e.Repository.FullName)
	case *githook.PushEvent:
		log.Printf(`event="Received hook event" type="push" repo="%s"`, e.Repository.FullName)
		for _, commit := range e.Commits {
			// Warning: Be careful if you try to run this under a lambda type
			// (e.g. deploying with apex) as these go routines will probably be
			// killed as soon as the parent function returns.
			go checkCommit(e.Repository.FullName, commit.ID)
		}
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

	defer resp.Body.Close() // TODO not needed?

	if resp.StatusCode != http.StatusOK {
		// TODO Private repositories not supported yet
		log.Printf(`event="Commit non-existent or on private repo" %s status_code="%d"`, bindLog, resp.StatusCode)
		// TODO need to determine what to do if this fails
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf(`event="Failed to retrieve commit info" %s error="%v"`, bindLog, err)
		// TODO need to determine what to do if this fails
		return
	}

	// TODO More robust checking that info returned is valid
	// ...

	var c githook.Commit
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

	return

	// TODO - Need to actually do something sensible with the reports - like alert someone!
	// TODO - Though splunk et al could be used in the short term to alert on logged above
}
