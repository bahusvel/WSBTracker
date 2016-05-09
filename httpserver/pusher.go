package main

import (
	"encoding/json"
	"net/http"
	"bytes"
	//"fmt"
	"io/ioutil"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"
)

type Response struct {
	MulticastID  int64    `json:"multicast_id"`
	Success      int      `json:"success"`
	Failure      int      `json:"failure"`
	CanonicalIDs int      `json:"canonical_ids"`
	Results      []Result `json:"results"`
}

type Result struct {
	MessageID      string `json:"message_id"`
	RegistrationID string `json:"registration_id"`
	Error          string `json:"error"`
}

type PushNotificaiton struct {
	To		string
	Title 	string
	Message	string
}

const API_KEY string = ""

func newNotificaiton(title string, body string, content bool, to string) ([]byte, error){
	notifbody := map[string]interface{}{"title": title, "body":body, "sound":"default"}
	full := map[string]interface{}{"to": to, "notification": notifbody, "content_available":content}
	b, err := json.Marshal(full)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func sendNotification(data []byte, client *http.Client) (*Response, error) {
	req, _ := http.NewRequest("POST", "https://gcm-http.googleapis.com/gcm/send", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "key="+API_KEY)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	err = json.Unmarshal(contents, &response)
	return &response, err
}

func gcmPush(ctx context.Context, message *PushNotification) error {
	client := urlfetch.Client(ctx)
	notifs, err := newNotificaiton(message.Title, message.Body, true, message.To)
	if err != nil {
		return err
	}
	sendNotification(notifs, client)
	return nil
}

func main(){
	testNotification()
}
