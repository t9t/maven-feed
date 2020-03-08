package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/feeds"
)

var (
	debugEnabled = false
	selfUrl      = ""
	maxResults   = 20
)

var artifactSpecs []ArtifactSpec

func main() {
	bindHost := envOrDefault("BIND_HOST", "0.0.0.0")
	bindPort := envOrDefault("BIND_PORT", "8080")
	artifactSpecsString := mustEnv("ARTIFACTS")
	selfUrl = mustEnv("SELF_URL")
	debugEnabled = os.Getenv("DEBUG_ENABLED") == "true"

	var err error
	artifactSpecs, err = parseArtifactSpecs(artifactSpecsString)
	if err != nil {
		log.Fatal(err)
	}

	logDebug("Parsed artifact specs: %+v\n", artifactSpecs)

	http.HandleFunc("/rss", rss)
	http.HandleFunc("/atom", atom)
	http.HandleFunc("/json", jsonFeed)

	hostAndPort := bindHost + ":" + bindPort
	log.Println("Listening on " + hostAndPort)
	log.Fatal(http.ListenAndServe(hostAndPort, nil))
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Environment variable %s not set or empty", key)
	}
	return v
}

func envOrDefault(key string, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func parseArtifactSpecs(s string) ([]ArtifactSpec, error) {
	items := strings.Split(s, "|")
	r := make([]ArtifactSpec, len(items))
	for i, item := range items {
		groupAndArtifact := strings.Split(item, ":")
		if len(groupAndArtifact) != 2 {
			return nil, errors.New("invalid artifact specification format")
		}
		r[i] = ArtifactSpec{Group: groupAndArtifact[0], Name: groupAndArtifact[1]}
	}
	return r, nil
}

func rss(w http.ResponseWriter, r *http.Request) {
	produceFeed(w, r, false, "application/rss+xml", (*feeds.Feed).ToRss)
}

func atom(w http.ResponseWriter, r *http.Request) {
	produceFeed(w, r, true, "application/atom+xml", (*feeds.Feed).ToAtom)
}

func jsonFeed(w http.ResponseWriter, r *http.Request) {
	produceFeed(w, r, false, "application/json", (*feeds.Feed).ToJSON)
}

func produceFeed(w http.ResponseWriter, r *http.Request, includeAuthor bool, contentType string, toFeedFunc func(*feeds.Feed) (string, error)) {
	artifacts, err := downloadAllArtifacts()
	if err != nil {
		log.Printf("Error fetching artifacts: %v\n", err)
		writeError(w)
		return
	}

	sorted := sortArtifactsByTimestampDesc(artifacts)

	feed := &feeds.Feed{
		Title:       "Maven Libraries Versions",
		Link:        &feeds.Link{Href: selfUrl},
		Description: "Lists versions of Maven libraries",
		Updated:     time.Now(),
		Items:       artifactsToFeedItems(sorted, includeAuthor),
	}

	output, err := toFeedFunc(feed)
	if err != nil {
		log.Printf("Error converting feed: %v\n", err)
		writeError(w)
		return
	}
	w.Header().Set("Content-Type", contentType)
	fmt.Fprint(w, output)
}

func downloadAllArtifacts() ([]Artifact, error) {
	r := make([]Artifact, 0)
	for _, spec := range artifactSpecs {
		artifacts, err := fetchArtifacts(spec.Group, spec.Name, maxResults)
		if err != nil {
			return nil, err
		}
		r = append(r, artifacts...)
	}
	return r, nil
}

func writeError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	http.Error(w, "Internal error occurred producing feed", http.StatusInternalServerError)
}

func logDebug(format string, args ...interface{}) {
	if debugEnabled {
		log.Printf(format, args...)
	}
}

func artifactsToFeedItems(artifacts []Artifact, includeAuthor bool) []*feeds.Item {
	items := make([]*feeds.Item, len(artifacts))
	for i, a := range artifacts {
		tm := time.Unix(a.Timestamp/1000, 0)
		desc := fmt.Sprintf("New artifact version: groupId: %s; artifactId: %s; version: %s", a.Group, a.Name, a.Version)
		items[i] = &feeds.Item{
			Id:          fmt.Sprintf("%s:%s:%s", a.Group, a.Name, a.Version),
			Title:       fmt.Sprintf("%s:%s:%s", a.Group, a.Name, a.Version),
			Link:        &feeds.Link{Href: fmt.Sprintf("https://search.maven.org/artifact/%s/%s/%s/jar", a.Group, a.Name, a.Version)},
			Description: desc,
			Content:     desc,
			Created:     tm,
			Updated:     tm,
		}
		if includeAuthor {
			items[i].Author = &feeds.Author{Name: a.Group}
		}
	}

	return items
}

func fetchArtifacts(groupId string, artifactId string, maxResults int) ([]Artifact, error) {
	url := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=g:%%22%s%%22+AND+a:%%22%s%%22&core=gav&rows=%d&wt=json", groupId, artifactId, maxResults)
	logDebug("Fetching artifacts for %s:%s from: %s\n", groupId, artifactId, url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching data for %s:%s: %v", groupId, artifactId, err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body for %s:%s: %v", groupId, artifactId, err)
	}

	if debugEnabled {
		log.Printf("Received body for %s:%s: %v\n", groupId, artifactId, string(body))
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling data for %s:%s: %v", groupId, artifactId, err)
	}

	logDebug("Parsed artifacts for %s:%s to: %+v", groupId, artifactId, response)
	if len(response.Data.Artifacts) == 0 {
		log.Printf("No artifacts received for %s:%s\n", groupId, artifactId)
	}
	return response.Data.Artifacts, nil
}

func sortArtifactsByTimestampDesc(artifacts []Artifact) []Artifact {
	sorted := make([]Artifact, len(artifacts))
	copy(sorted, artifacts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp > sorted[j].Timestamp
	})
	return sorted
}

type Response struct {
	Data ResponseData `json:"response"`
}

type ResponseData struct {
	Artifacts []Artifact `json:"docs"`
}

type Artifact struct {
	Group     string `json:"g"`
	Name      string `json:"a"`
	Version   string `json:"v"`
	Timestamp int64  `json:"timestamp"`
}

type ArtifactSpec struct {
	Group string
	Name  string
}
