package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2/bson"
)

const (
	emailCollection = "emails"
)

type Email struct {
	DraftID       string   `bson:"draft_id"`
	Owner         string   `bson:"owner"`
	Collaborators []string `bson:"collaborators"`
	Edits         []Edit   `bson:"edits"`
}

type Edit struct {
	Editor  string `bson:"editor"`
	Content string `bson:"content"`
}

type newEmailRequest struct {
	DraftID string `json:draft_id`
}

// newEmail is an API endpoint to create a new Draft object
func newEmail(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var newDraft newEmailRequest
	buf, err := ioutil.ReadAll(r.Body) // TODO: fix
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Failed to read in request body => {%s}", err)
		return
	}
	defer r.Body.Close()

	// decode the request
	err = json.Unmarshal(buf, &newDraft)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Failed to decode JSON request => {%s}, data => {%s}", err, string(buf))
		return
	}

	log.Printf("Draft recieved => %#v", newDraft)

	// see if the draft alread exists
	n, err := mgoConn.C(emailCollection).Find(bson.M{"draft_id": newDraft.DraftID}).Count()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Failed to check database for existance => {%s}", err)
		return
	} else if n != 0 {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("tried to recreate object")
		return
	}

	// grab session reference
	s, err := store.Get(r, sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Failed to access the session => {%s}", err)
		return
	}

	// get actual Gmail draft
	body, err := getDraft(r, newDraft.DraftID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("Failed to access the gmail draft => {%s}", err)
		return
	}

	// insert the new draft
	owner := s.Values[userEmailKey].(string)
	mail := Email{
		DraftID:       newDraft.DraftID,
		Owner:         owner,
		Collaborators: []string{owner},
		Edits: []Edit{
			Edit{
				Editor:  owner,
				Content: body,
			},
		},
	}
	err = mgoConn.C(emailCollection).Insert(&mail)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("failed to insert new draft to %s=> {%s}", emailCollection, err)
		return
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

	err = mgoConn.C(emailCollection).Update(
		bson.M{"draftID": draftID},
		bson.M{"$push": bson.M{"Edits": &change}})
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "failed to insert new draft => {%s}", err)
		log.Printf("failed to insert new draft => {%s}", err)
	}

	// TODO: push update to Gmail
}

var listProjection bson.M = bson.M{
	"DraftID": 1,
	"Owner":   1,
}

type listSummary struct {
	DraftID string `json:"draft_id"`
	Owner   string `json:""`
}

// listAvailable returns a list of all available drafts
func listAvailable(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// get the requester from the session
	s, err := store.Get(r, sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to access the session => {%s}", err)
		return
	}
	currentUser := s.Values[userEmailKey].(string)

	// find all drafts the user has access to
	drafts := []listSummary{}
	err = mgoConn.C(emailCollection).Find(
		bson.M{"Collaborators": currentUser},
	).Select(listProjection).All(&drafts)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed run mongo query => {%s}", err)
		return
	}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(drafts)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed write data to conn => {%s}", err)
		return
	}
}
