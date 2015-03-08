package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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
func checkIfAuthenticated(h func(http.ResponseWriter, *http.Request, httprouter.Params)) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
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

		h(w, r, p)
	})
}

func hi(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	log.Printf("DEBUG => %s: %s", r.Method, r.URL.Path)
}

func main() {
	router := httprouter.New()

	router.GET("/", hi)
	router.POST("/authorize", handleAuthorize)
	router.GET("/authenticate", needAuth)
	router.GET("/list", checkIfAuthenticated(list_emails))

	// API
	router.POST("/draft/create", checkIfAuthenticated(newEmail))
	router.GET("/draft/list", checkIfAuthenticated(listAvailable))
	router.POST(fmt.Sprintf("/draft/id/:%s", draftIDParam), checkIfAuthenticated(draftUpdate))

	//Google will redirect to this page to return your code, so handle it appropriately
	router.GET("/oauth2callback", handleOAuth2Callback)

	router.NotFound = debugLog

	err := http.ListenAndServe(":"+serverPort, context.ClearHandler(router))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
