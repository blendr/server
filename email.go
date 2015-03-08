package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
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

// draftUpdate
func draftUpdate(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	draftID := p.ByName(draftIDParam)
	var change Edit

	// decode the request into an Edit
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&change)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to decode JSON request => {%s}", err)
	}

	// add the author to the change
	s, err := store.Get(r, sessionKey)
	if err != nil {
		fmt.Fprintf(w, "Failed to access the session => {%s}", err)
		return
	}
	change.Editor = s.Values[userEmailKey].(string)

	mgoConn.C(emailCollection).Update(
		bson.M{"draftID": draftID},
		bson.M{"$push": bson.M{"Edits": &change}})
}
