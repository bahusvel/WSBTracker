package httpserver

import (
	"google.golang.org/appengine/log"
	"golang.org/x/net/context"
	"math"
	"encoding/json"
	"net/http"
	"bytes"
	"io/ioutil"
	"google.golang.org/appengine/urlfetch"
)

const (
	// According to Wikipedia, the Earth's radius is about 6,371km
	EARTH_RADIUS = 6371 * 1000
)

func checkTriggers(ctx context.Context, busPosition Position) {
	triggers := getGeoTriggers(ctx, 0, 0); // near arguments are ignored
	for _, trigger := range triggers {
		if trigger.distanceTo(busPosition.Latitude, busPosition.Longitude) < 20 {
			log.Infof(ctx, "Going to notify these people: ", trigger.NotifyDrivers)
			for _, driverEmail := range trigger.NotifyDrivers {
				driver, _ := getDriver(ctx, driverEmail)
				if driver != nil && driver.PushToken != ""{
					notification := PushNotificaiton{To: driver.PushToken, Title:"Bus Arrived", Message:"Bus Arrived"}
					gcmPush(ctx, &notification)
				}
			}
		}
	}
}

// Calculates the Haversine distance between two points in meters.
// Original Implementation from: http://www.movable-type.co.uk/scripts/latlong.html
func (p *GeoTrigger) distanceTo(latitude2 float64, longitude2 float64) float64 {
	dLat := (latitude2 - p.Latitude) * (math.Pi / 180.0)
	dLon := (longitude2 - p.Longitude) * (math.Pi / 180.0)

	lat1 := p.Latitude * (math.Pi / 180.0)
	lat2 := latitude2 * (math.Pi / 180.0)

	a1 := math.Sin(dLat/2) * math.Sin(dLat/2)
	a2 := math.Sin(dLon/2) * math.Sin(dLon/2) * math.Cos(lat1) * math.Cos(lat2)

	a := a1 + a2

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EARTH_RADIUS * c
}

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

func newNotification(title string, body string, content bool, to string) ([]byte, error){
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
	if API_KEY == "" {
		panic("YOUR API KEY IS EMPTY YOU NUB!!!")
	}
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

func gcmPush(ctx context.Context, message *PushNotificaiton) error {
	client := urlfetch.Client(ctx)
	notifs, err := newNotification(message.Title, message.Message, true, message.To)
	if err != nil {
		return err
	}
	log.Infof(ctx, "Sending notification to", message.To)
	response, err := sendNotification(notifs, client)
	log.Infof(ctx, "Result of sending is: ", response, err)
	return nil
}