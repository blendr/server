package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"

	"code.google.com/p/goauth2/oauth"
	"code.google.com/p/google-api-go-client/plus/v1"
	"github.com/google/google-api-go-client/gmail/v1"
	"github.com/gorilla/sessions"
)

type appError struct {
	Err     error
	Message string
	Code    int
}

const (
	applicationName = "Gmail Peer Edit"
	sessionKey      = "gmail-code"
)

var (
	serverPort   string
	clientID     = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	// store initializes the Gorilla session store.
	store = sessions.NewCookieStore([]byte("qwerty123")) // TODO: configure

	oauthClient *http.Client
)

func init() {
	serverPort = os.Getenv("PORT")
	if serverPort == "" {
		log.Fatal("No value found in environment for PORT")
	}

	/*
		// Your credentials should be obtained from the Google
		// Developer Console (https://console.developers.google.com).
		conf := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  "http://localhost:5000/oauth2callback",
			Scopes: []string{
				gmail.MailGoogleComScope,
			},
			Endpoint: google.Endpoint,
		}
		// Redirect user to Google's consent page to ask for permission
		// for the scopes specified above.
		url := conf.AuthCodeURL("state")
		fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

		// Handle the exchange code to initiate a transport.
		tok, err := conf.Exchange(oauth2.NoContext, "authorization-code")
		if err != nil {
			log.Fatal(err)
		}

		oauthClient = conf.Client(oauth2.NoContext, tok)
	*/
}

// indexTemplate is the HTML template we use to present the index page.
var indexTemplate = template.Must(template.ParseFiles("index.html"))

// index sets up a session for the current user and serves the index page
func index(w http.ResponseWriter, r *http.Request) {
	// This check prevents the "/" handler from handling all requests by default
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Create a state token to prevent request forgery and store it in the session
	// for later validation
	session, err := store.Get(r, "sessionName")
	if err != nil {
		log.Println("error fetching session:", err)
		// Ignore the initial session fetch error, as Get() always returns a
		// session, even if empty.
		//return &appError{err, "Error fetching session", 500}
	}
	state := randomString(64)
	session.Values["state"] = state
	session.Save(r, w)

	stateURL := url.QueryEscape(session.Values["state"].(string))

	// Fill in the missing fields in index.html
	var data = struct {
		ApplicationName, ClientID, State string
	}{applicationName, clientID, stateURL}

	// Render and serve the HTML
	err = indexTemplate.Execute(w, data)
	if err != nil {
		log.Println("error rendering template:", err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "error rendering template => {%s}", err)
	}
}

// people fetches the list of people user has shared with this app
func people(w http.ResponseWriter, r *http.Request) *appError {
	session, err := store.Get(r, "sessionName")
	if err != nil {
		log.Println("error fetching session:", err)
		return &appError{err, "Error fetching session", 500}
	}
	token := session.Values["accessToken"]
	// Only fetch a list of people for connected users
	if token == nil {
		m := "Current user not connected"
		return &appError{errors.New(m), m, 401}
	}

	// Create a new authorized API client
	t := &oauth.Transport{Config: oauthCfg}
	tok := new(oauth.Token)
	tok.AccessToken = token.(string)
	t.Token = tok
	service, err := plus.New(t.Client())
	if err != nil {
		return &appError{err, "Create Plus Client", 500}
	}

	// Get a list of people that this user has shared with this app
	people := service.People.List("me", "visible")
	peopleFeed, err := people.Do()
	if err != nil {
		m := "Failed to refresh access token"
		if err.Error() == "AccessTokenRefreshError" {
			return &appError{errors.New(m), m, 500}
		}
		return &appError{err, m, 500}
	}
	w.Header().Set("Content-type", "application/json")
	err = json.NewEncoder(w).Encode(&peopleFeed)
	if err != nil {
		return &appError{err, "Convert PeopleFeed to JSON", 500}
	}
	return nil
}

func list_emails(w http.ResponseWriter, r *http.Request) {
	gservice, err := gmail.New(oauthClient)
	if err != nil {
		log.Fatalf("Failed to create new gmail service => %s", err.Error())
	}

	call := gservice.Users.Messages.List("me")
	resp, err := call.Do()
	if err != nil {
		log.Fatalf("Failed to query gmail for email list => %s", err.Error())
	}

	fmt.Fprintf(w, "<h1>emails</h1>")
	for _, m := range resp.Messages {
		fmt.Fprintf(w, m.Id+"<br>")
	}
}

// randomString returns a random string with the specified length
func randomString(length int) (str string) {
	b := make([]byte, length)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/list", list_emails)

	//Google will redirect to this page to return your code, so handle it appropriately
	http.HandleFunc("/oauth2callback", handleOAuth2Callback)

	err := http.ListenAndServe(":"+serverPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
