package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mmcdole/gofeed"
	"github.com/op/go-logging"
	yaml "gopkg.in/yaml.v2"
)

//Feed contains all basic information of a feed, used for saving and interpretation
type Feed struct {
	Title       string
	Description string
	Link        string
	Items       []Article
}

//Article contains all article specific informations
type Article struct {
	Title           string
	Description     string
	Content         string
	Author          AuthorInfo
	Categories      []string
	Link            string
	PublishedParsed string
	URL             string
}

//AuthorInfo to comply with json structure returned by go-parser
type AuthorInfo struct {
	Name string
}

//FeedRegistry structure to save and interpret saved Feed information
type FeedRegistry struct {
	Name string
	URL  string
}

type configFile struct {
	Feeds []FeedRegistry
	port  string
}

type token struct {
	Token string
}

/******************* global variables */
var config = configFile{}
var feeds = []FeedRegistry{}

/******************* logger init */
var log = logging.MustGetLogger("authentication")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc}|%{shortfile} : %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func main() {
	loggerInitilization()
	verifyFeeds()
	go updateFeed()
	r := mux.NewRouter()
	//This will allow access to the server even if Request originated somewhere else
	allowOrigins := handlers.AllowedOrigins([]string{"*"})
	allowMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "OPTIONS", "HEAD"})
	allowHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	r.HandleFunc("/feed/{name}", getFeed)
	r.HandleFunc("/feed/{name}/{id}", getArticle)
	r.HandleFunc("/feeds", getFeeds)
	// r.HandleFunc("/addFeed", putFeeds)
	http.Handle("/", r)
	log.Notice("Listening on port 8000")
	log.Fatal(http.ListenAndServe(":8000", handlers.CORS(allowOrigins, allowMethods, allowHeaders)(r)))
}

func getFeed(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Error while reading in body")
	}
	tok := token{}
	json.Unmarshal([]byte(body), &tok)
	if verifyToken(tok.Token) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
		return
	}
	vars := mux.Vars(r)
	name, _ := vars["name"]
	feed, _ := ioutil.ReadFile(name)
	w.Header().Set("Content-Type", "application/json")
	w.Write(feed)
}

/*	Returns article of id from requested feed
	Input: id: string, name: string
	Output: article: json-string
*/
func getArticle(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Error while reading in body")
	}
	tok := token{}
	json.Unmarshal([]byte(body), &tok)
	if verifyToken(tok.Token) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
		return
	}
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])
	name, _ := vars["name"]
	res := &Feed{}
	feed, _ := ioutil.ReadFile(name)
	json.Unmarshal([]byte(feed), res)
	article := res.Items[id]
	js, _ := json.Marshal(article)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

/*	update background function, recommended to be called as go routine
	Input: none
	Output: none
*/
func updateFeed() {
	catchFeeds()
	ticker := time.NewTicker(10800 * time.Second)
	for range ticker.C {
		go catchFeeds()
	}
}

/*	Periodically called function that updates content of all Feeds
	Input: none
	Output: none
*/
func catchFeeds() {
	log.Notice("Routine Background Task: Catching Feeds")
	parser := gofeed.NewParser()
	for i := 0; i < len(feeds); i++ {
		cont, err := parser.ParseURL(feeds[i].URL)
		if err != nil {

			log.Error("Connection could not be established. Parser returned error:", err)
			time.Sleep(20 * time.Second)
			log.Info("Trying alternative")
			httpresponse, err := http.Get(feeds[i].URL)
			if err != nil {
				log.Error("Connection could not be established again. Parser returned error:", err)
			}

			defer httpresponse.Body.Close()
			XMLString, _ := ioutil.ReadAll(httpresponse.Body)
			cont, err = parser.ParseString(string(XMLString))
			if err != nil {
				log.Error("Connection could not be established again. Parser returned error:", err)
			} else {
				f, _ := os.Create(feeds[i].Name)
				js, _ := json.Marshal(cont)
				res := &Feed{}
				json.Unmarshal([]byte(js), res)
				save, _ := json.Marshal(res)
				f.Write(save)
				defer f.Close()

			}
		} else {
			f, _ := os.Create(feeds[i].Name)
			js, _ := json.Marshal(cont)
			res := &Feed{}
			json.Unmarshal([]byte(js), res)
			save, _ := json.Marshal(res)
			f.Write(save)
			defer f.Close()
		}
	}
}

/*	Returns all known Feeds
	Input: none
	Output: feeds: json-string
*/
func getFeeds(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("Error while reading in body")
	}
	tok := token{}
	json.Unmarshal([]byte(body), &tok)
	if verifyToken(tok.Token) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 - Forbidden"))
		return
	}
	js, err := json.Marshal(feeds)
	if err != nil {
		log.Notice("Marshaling failed:", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// func putFeeds(w http.ResponseWriter, r *http.Request) {
// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		log.Error("Error while reading in body")
// 	}
// 	tok := token{}
// 	json.Unmarshal([]byte(body), &tok)
// 	if verifyToken(tok.Token) == false {
// 		w.WriteHeader(http.StatusForbidden)
// 		w.Write([]byte("403 - Forbidden"))
// 		return
// 	}
//
// 	res := FeedRegistry{}
// 	body, _ = ioutil.ReadAll(r.Body)
// 	json.Unmarshal([]byte(body), &res)
// 	f, _ := os.OpenFile("defaults", os.O_APPEND|os.O_WRONLY, 0644)
// 	feed := FeedRegistry{
// 		Name: "test",
// 		URL:  res.URL,
// 	}
// 	js, _ := json.Marshal(feed)
// 	f.Write(js)
// 	defer f.Close()
// }

/*	Initial function that creates the defaults file which contains all known Feeds
	Input: none
	Output: none
*/
func verifyFeeds() {
	log.Notice("Looking for Feed")
	_, err := os.Stat("config/feeds.yml")
	if err != nil {
		log.Notice("No Config found")
		log.Notice("Initilizating Feed List")
		os.MkdirAll("config", 0722)

		f, err := os.Create("config/feeds.yml")

		if err != nil {
			log.Error("Creating file found:", err)
		}

		defEntries := []FeedRegistry{
			FeedRegistry{
				Name: "SpaceFlightNow",
				URL:  "https://spaceflightnow.com/feed/",
			},
			FeedRegistry{
				Name: "The Guardian",
				URL:  "https://www.theguardian.com/international/rss",
			},
			FeedRegistry{
				Name: "Heise Online",
				URL:  "https://www.heise.de/newsticker/heise-atom.xml",
			},
			FeedRegistry{
				Name: "reddit",
				URL:  "https://www.reddit.com/.rss",
			},
			FeedRegistry{
				Name: "New York Times",
				URL:  "http://rss.nytimes.com/services/xml/rss/nyt/World.xml",
			},
		}

		yml, _ := yaml.Marshal(defEntries)
		feeds = defEntries
		f.Write(yml)
		defer f.Close()
	} else {
		log.Notice("Using existing config")
		f, err := ioutil.ReadFile("config/feeds.yml")
		if err != nil {
			log.Error("Reading in failed:", err)
		}
		yaml.Unmarshal(f, &feeds)

	}

}

func verifyToken(token string) (validity bool) {
	js := []byte(`{"Token":"` + token + `"}`)
	req, err := http.NewRequest("POST", "http://localhost:9000/checkToken", bytes.NewBuffer(js))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.Status == "200 OK" {
		validity = true
	} else {
		validity = false
	}
	return
}

func loggerInitilization() {
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Formatter := logging.NewBackendFormatter(backend1, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
	logging.SetBackend(backend1Leveled, backend1Formatter)
}
