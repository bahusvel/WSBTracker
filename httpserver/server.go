package httpserver

import (
    "fmt"
    "net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
	"google.golang.org/appengine/log"

	// storage
	"io/ioutil"
	"encoding/json"
	"golang.org/x/net/context"
	"strconv"
	"time"
)

type PositionEntry struct {
	Position
	UserEmail 	string
}

type Position struct {
	Latitude	float64
	Longitude	float64
}

type GenericResponse struct {
	Reponse		string
}

func init() {
    http.HandleFunc("/", home)
	http.HandleFunc("/position/log", logPosition)
}

func writeResponse(w http.ResponseWriter, errmessege string){
	res, _ := json.Marshal(GenericResponse{Reponse:errmessege})
	w.Write(res)
}

func pos2DB(pEntry PositionEntry, ctx context.Context){
	ctime := int(time.Now().Unix())
	k := datastore.NewKey(ctx, "PositionEntry", pEntry.UserEmail+":"+strconv.Itoa(ctime), 0, nil)
	if _, err := datastore.Put(ctx, k, &pEntry); err != nil {
		log.Errorf(ctx, "Failed to put in datastore %v", err)
	}
}

func logPosition(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		writeResponse(w, "Unauthorized")
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeResponse(w, "Unreadable Request")
		return
	}
	position := &Position{}
	if json.Unmarshal(body, position) != nil {
		writeResponse(w, "Unreadable Request")
		return
	}
	pEntry := PositionEntry{Position:*position, UserEmail:u.Email}
	pos2DB(pEntry, ctx)
	writeResponse(w, "Success")
	log.Infof(ctx, "Position Store Successful, latitude: %f, longitude: %f", position.Latitude, position.Longitude)
}

func home(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

