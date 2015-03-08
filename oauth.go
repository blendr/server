package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"google.golang.org/api/gmail/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var notAuthenticatedTemplate = template.Must(template.New("").Parse(`
<html>
  <body>
    You have currently not given permissions to access your data. Please authenticate this app with the Google OAuth provider.
    <form action="/authorize" method="POST"><input type="submit" value="Ok, authorize this app with my id"/></form>
  </body>
</html>
`))

// var oauthCfg = &oauth.Config{
var oauthCfg = &oauth2.Config{
	ClientID:     clientID,
	ClientSecret: clientSecret,
	Endpoint:     google.Endpoint,
	// To return your oauth2 code, Google will redirect the browser to this page that you have defined
	// TODO: This exact URL should also be added in your Google API console for this project
	// within "API Access"->"Redirect URIs"
	RedirectURL: "http://localhost:5000/oauth2callback",
	// This is the 'scope' of the data that you are asking the user's permission to access.
	// For getting user's info, this is the url that Google has defined.
	Scopes: []string{
		gmail.MailGoogleComScope,
	},
}

func needAuth(w http.ResponseWriter, r *http.Request) {
	notAuthenticatedTemplate.Execute(w, nil)
}

// Start the authorization process
func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	//Get the Google URL which shows the Authentication page to the user
	url := oauthCfg.AuthCodeURL("")
	//redirect user to that page
	http.Redirect(w, r, url, http.StatusFound)
}

// Function that handles the callback from the Google server
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request) {
	//Get the code from the response
	code := r.FormValue("code")

	// Exchange the received code for a token
	// tok, err := oauthCfg.Exchange(oauth2.NoContext, code)

	s, err := store.New(r, sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to create new session => {%s}", err)
		return
	}

	s.Values[codeKey] = code

	tok, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Printf("failed to exchange with code => {%s}", err)
		return
	}
	s.Values[tokenKey] = tok.AccessToken

	store.Save(r, w, s)
	fmt.Fprintf(w, "You're now authenticated!<br><a href=\"/\">home</a>")

	/*
		gservice, err := gmail.New(t.Client())
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
	*/
}

// makeClient creates an oauth2 client from a session variable
func makeClient(r *http.Request) *http.Client {
	session, err := store.Get(r, sessionKey)
	if err != nil {
		log.Printf("Failed to find session => {%s}")
		return nil
	}

	// Exchange the received code for a token
	code := session.Values[codeKey].(string)
	log.Printf("Code => {%s}", code)

	/*
		tok, err := oauthCfg.Exchange(oauth2.NoContext, code)
		if err != nil {
			log.Printf("failed to exchange with code => {%s}", err)
			return nil
		}
	*/

	tok := &oauth2.Token{
		AccessToken: session.Values[tokenKey].(string),
	}

	return oauthCfg.Client(oauth2.NoContext, tok)
}
