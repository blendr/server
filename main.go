package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/google-api-go-client/gmail/v1"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
)

type appError struct {
	Err     error
	Message string
	Code    int
}

const (
	applicationName = "Gmail Peer Edit"

	sessionKey = "blendr"
	codeKey    = "gmail-code"
	tokenKey   = "gmail-token"
)

var (
	serverPort   string
	clientID     = os.Getenv("GOOGLE_CLIENT_ID")
	clientSecret = os.Getenv("GOOGLE_CLIENT_SECRET")

	// store initializes the Gorilla session store.
	store = sessions.NewCookieStore([]byte("qwerty123")) // TODO: configure
)

func init() {
	serverPort = os.Getenv("PORT")
	if serverPort == "" {
		log.Fatal("No value found in environment for PORT")
	}
}

func checkIfAuthenticated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionKey)
		if err != nil {
			log.Printf("error getting session => {%s}", err)
			return
		}

		_, exists := session.Values[codeKey]
		log.Printf("%#v", session.Values) // TODO: delete

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

// randomString returns a random string with the specified length
func randomString(length int) (str string) {
	b := make([]byte, length)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func hi(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<h1>hi</h1><a href="/list">list emails</a>`)
}

func main() {
	http.HandleFunc("/", hi)
	http.HandleFunc("/authorize", handleAuthorize)
	http.HandleFunc("/authenticate", needAuth)
	http.Handle("/list", checkIfAuthenticated(http.HandlerFunc(list_emails)))

	//Google will redirect to this page to return your code, so handle it appropriately
	http.HandleFunc("/oauth2callback", handleOAuth2Callback)

	err := http.ListenAndServe(":"+serverPort, context.ClearHandler(http.DefaultServeMux))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
