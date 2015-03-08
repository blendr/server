package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2/bson"
)

const (
	emailCollection = "emails"
)

type Email struct {
	DraftID       string
	Owner         string
	Collaborators []string
	Edits         []Edit
}

type Edit struct {
	Editor  string
	Content string
}

type newEmailRequest struct {
	DraftID string `json:draft_id`
}

// newEmail is an API endpoint to create a new Draft object
func newEmail(w http.ResponseWriter, r *http.Request) {
	var newDraft newEmailRequest

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	// decode the request
	err := decoder.Decode(&newDraft)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to decode JSON request => {%s}", err)
	}

	// see if the draft alread exists
	n, err := mgoConn.C(emailCollection).Find(bson.M{"draftID": newDraft.DraftID}).Count()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to check database for existance => {%s}", err)
	} else if n != 0 {
		fmt.Fprintf(w, "tried to recreate object")
		return
	}

	// insert the new draft
	err = mgoConn.C(emailCollection).Insert(&newDraft)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to add to database => {%s}", err)
	}
}
