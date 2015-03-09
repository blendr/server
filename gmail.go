package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"google.golang.org/api/gmail/v1"
)

func listEmails(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func getDraft(r *http.Request, draftID string) (string, error) {
	client := makeClient(r)
	if client == nil {
		return "", fmt.Errorf("Error while creating oauth2 client")
	}

	gservice, err := gmail.New(client)
	if err != nil {
		log.Fatalf("Failed to create new gmail service => %s", err.Error())
	}

	// grab session reference
	s, err := store.Get(r, sessionKey)
	if err != nil {
		return "", fmt.Errorf("Failed to access the session => {%s}", err)
	}

	uds := gmail.NewUsersDraftsService(gservice)
	// TODO: print out all ID's to see if it even has the same shape?

	draft, err := uds.Get(s.Values[userIDKey].(string), draftID).Do()
	if err != nil {
		return "", fmt.Errorf("Failed to access draft => {%s}", err)
	}

	return draft.Message.Raw, nil
}
