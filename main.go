package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/google-api-go-client/gmail/v1"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"

	"gopkg.in/mgo.v2"
)

const (
	applicationName = "Gmail Peer Edit"
	sessionKey      = "blendr"
	codeKey         = "gmail-code"
	tokenKey        = "gmail-token"
	userEmailKey    = "gmail-email"
	draftIDParam    = "draft_id_param"
)

var (
	serverPort   string
	clientID     = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	mongoDatabase = "blendr"

	// store initializes the Gorilla session store.
	store = sessions.NewCookieStore([]byte("qwerty1234")) // TODO: configure

	// mgoConn is the connection to mongodb
	mgoConn *mgo.Database
)

func init() {
	serverPort = os.Getenv("PORT")
	if serverPort == "" {
		log.Fatal("No value found in environment for PORT")
	}

	mgoSession, err := mgo.Dial("localhost:27017")
	if err != nil {
		log.Fatalf("Cannot connect to Mongo => {%s}", err)
	}
	// durable writes
	mgoSession.SetSafe(&mgo.Safe{})

	// connect to the right database
	mgoConn = mgoSession.DB(mongoDatabase)
}

// checkIfAuthenticated handles checking if the token is in the cookie. If it is not
// then we redirect to let the user re-authenticate.
func checkIfAuthenticated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionKey)
		if err != nil {
			log.Printf("error getting session => {%s}", err)
			return
		}

		// see if the key is there
		_, exists := session.Values[tokenKey]
		if !exists {
			http.Redirect(w, r, "/authenticate", http.StatusSeeOther)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func list_emails(w http.ResponseWriter, r *http.Request) {
	client := makeClient(r)
	if client == nil {
		fmt.Fprintf(w, "Error while creating oauth2 client")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	gservice, err := gmail.New(client)
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

func hi(w http.ResponseWriter, r *http.Request) {
	s, err := store.Get(r, sessionKey)
	if err != nil {
		fmt.Fprintf(w, "Failed to access the session => {%s}", err)
		return
	}

	user := s.Values[userEmailKey]
	if user == "" {
		user = " "
	}
	fmt.Fprintf(w, "<h1>hi %s</h1><a href=\"/list\">list emails</a>", user)
}

func debugLog(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s: %s", r.Method, r.URL.Path)
}

func main() {
	router := httprouter.New()

	router.HandlerFunc("GET", "/", hi)
	router.HandlerFunc("POST", "/authorize", handleAuthorize)
	router.HandlerFunc("GET", "/authenticate", needAuth)
	router.Handler("GET", "/list", checkIfAuthenticated(http.HandlerFunc(list_emails)))

	// API
	router.Handler("POST", "/draft/create", checkIfAuthenticated(http.HandlerFunc(newEmail)))
	router.POST(fmt.Sprintf("/draft/:%s", draftIDParam), draftUpdate)

	//Google will redirect to this page to return your code, so handle it appropriately
	router.HandlerFunc("GET", "/oauth2callback", handleOAuth2Callback)

	router.NotFound = debugLog

	err := http.ListenAndServe(":"+serverPort, context.ClearHandler(router))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
