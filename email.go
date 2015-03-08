package main

import (
	"encoding/json"
	"fmt"
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
func newEmail2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var newDraft newEmailRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	// decode the request
	err := decoder.Decode(&newDraft)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to decode JSON request => {%s}", err)
		return
	}

	// see if the draft alread exists
	n, err := mgoConn.C(emailCollection).Find(bson.M{"draftID": newDraft.DraftID}).Count()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to check database for existance => {%s}", err)
		return
	} else if n != 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "tried to recreate object")
		return
	}

	// insert the new draft
	owner := "natebrennand@gmail.com"
	body := "yolo test"
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
		fmt.Fprintf(w, "Failed to add to database => {%s}", err)
		return
	}
}

// newEmail is an API endpoint to create a new Draft object
func newEmail(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var newDraft newEmailRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	log.Printf("about to decode the request") // TODO: delete

	// decode the request
	err := decoder.Decode(&newDraft)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to decode JSON request => {%s}", err)
		return
	}

	fmt.Printf("about to check for an existing obj\n") // TODO: delete
	// see if the draft alread exists
	n, err := mgoConn.C(emailCollection).Find(bson.M{"draft_id": newDraft.DraftID}).Count()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to check database for existance => {%s}", err)
		return
	} else if n != 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "tried to recreate object")
		return
	}

	fmt.Printf("about to get the session/cookie\n") // TODO: delete
	// grab session reference
	s, err := store.Get(r, sessionKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to access the session => {%s}", err)
		return
	}

	fmt.Printf("about to get the gmail draft\n") // TODO: delete
	// get actual Gmail draft
	body, err := getDraft(r, newDraft.DraftID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to access the gmail draft => {%s}", err)
		return
	}
	fmt.Printf("email body => %s", body) // TODO: delete

	fmt.Printf("about to insert the new email draft\n") // TODO: delete
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
		fmt.Fprintf(w, "Failed to add to database => {%s}", err)
		return
	}
	fmt.Printf("done handing the request\n") // TODO: delete
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
