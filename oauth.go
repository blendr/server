package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	googleOauth "google.golang.org/api/oauth2/v2"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/plus/v1"

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
		plus.UserinfoEmailScope,
	},
}

func needAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	notAuthenticatedTemplate.Execute(w, nil)
}

// Start the authorization process
func handleAuthorize(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//Get the Google URL which shows the Authentication page to the user
	url := oauthCfg.AuthCodeURL("")
	//redirect user to that page
	http.Redirect(w, r, url, http.StatusFound)
}

// Function that handles the callback from the Google server
func handleOAuth2Callback(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//Get the code from the response
	code := r.FormValue("code")

	// access the session
	s, err := store.New(r, sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to create new session => {%s}", err)
		return
	}

	// add the code to regenerate a token to the cookie
	s.Values[codeKey] = code

	// createa token with the code
	tok, err := oauthCfg.Exchange(oauth2.NoContext, code)
	if err != nil {
		log.Printf("failed to exchange with code => {%s}", err)
		return
	}

	// stuff the token into the cookie
	s.Values[tokenKey], err = json.Marshal(tok)
	if err != nil {
		log.Printf("failed to marshal token to JSON => {%s}", err)
		return
	}

	// get the user's email and add it to the cookie
	client := oauthCfg.Client(oauth2.NoContext, tok)
	srv, err := googleOauth.New(client)
	if err != nil {
		log.Printf("failed to createa google oauth service => {%s}", err)
		return
	}
	callRes, err := googleOauth.NewUserinfoService(srv).Get().Do()
	if err != nil {
		log.Printf("failed to make call to google plus => {%s}", err)
		return
	}
	s.Values[userEmailKey] = callRes.Email

	// save the cookie and return
	store.Save(r, w, s)

	// redirect to the homepage
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// makeClient creates an oauth2 client from a session variable
func makeClient(r *http.Request) *http.Client {
	// grab the cookie
	session, err := store.Get(r, sessionKey)
	if err != nil {
		log.Printf("makeClient: Failed to find session => {%s}")
		return nil
	}

	tok := new(oauth2.Token)
	err = json.Unmarshal(session.Values[tokenKey].([]byte), tok)
	if err != nil {
		log.Printf("makeClient: Failed to unmarshal token => {%s}")
		return nil
	}

	// refresh token
	/*
		if !tok.Valid() {
			// createa token with the code
			newToken, err := oauthCfg.Exchange(oauth2.NoContext, session.Values[codeKey].(string))
			if err != nil {
				log.Printf("makeClient: failed to exchange with code => {%s}", err)
				return nil
			}

			// stuff the token into the cookie
			session.Values[tokenKey], err = json.Marshal(newToken)
			if err != nil {
				log.Printf("makeClient: failed to marshal token to JSON => {%s}", err)
				return nil
			}

			tok = newToken
		}
	*/

	return oauthCfg.Client(oauth2.NoContext, tok)
}
