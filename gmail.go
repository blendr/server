package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"google.golang.org/api/gmail/v1"
)

func list_emails(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
