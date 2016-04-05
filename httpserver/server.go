package httpserver

import (
    "fmt"
    "net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/user"
	"google.golang.org/appengine/log"
	"io/ioutil"
	"encoding/json"
	"strconv"
	//"golang.org/x/net/context"
)

type GenericResponse struct {
	Reponse		string
}

type DriverOperation struct{
	TheDriver 		Driver
	Operation 		string
}

type BusOperation struct {
	Bus			Bus
	Operation	string
}


func init() {
    http.HandleFunc("/", home)
	http.HandleFunc("/position/log", logPosition)
	http.HandleFunc("/busses/available", bussesAvailable)
	http.HandleFunc("/busses/drive", driveBus)
	http.HandleFunc("/busses/location", busLocation)
	http.HandleFunc("/admin/driver", adminDriver)
	http.HandleFunc("/admin/bus", adminBus)
	http.HandleFunc("/logout", logout)
}


func home(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

func logout(w http.ResponseWriter, r *http.Request) {
	url, _ := user.LogoutURL(appengine.NewContext(r), "/")
	http.Redirect(w, r, url, 301)
}

func busLocation(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	query := r.URL.Query()
	nbr, err := strconv.Atoi(query.Get("busNumber"))
	if err != nil {
		writeResponse(w, "Invalid Request")
		return
	}
	bus, bk, err := getBus(ctx, nbr, "unitec")
	if err != nil {
		writeResponse(w, "Bus Not Found")
		log.Errorf(ctx, "Bus Not Found")
		return
	}
	if bus.CurrentTrip == "" {
		writeResponse(w, "Bus Not Driving")
		log.Errorf(ctx, "Bus Not Driving")
		return
	}
	btk := datastore.NewKey(ctx, "BusTrip", bus.CurrentTrip, 0, bk)
	pq := datastore.NewQuery("Position").Ancestor(btk)
	var positions []Position
	ct, _ := pq.Count(ctx)
	pq.Offset(ct-1).Limit(1).GetAll(ctx, &positions)

	res, _ := json.Marshal(positions[0])
	w.Write(res)
}

func adminBus(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	busOp := &BusOperation{}
	if readRequest(r, busOp) != nil{
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}

	switch busOp.Operation {
	case "add":
		err := storeBus(ctx, &busOp.Bus, "unitec")
		if err != nil {
			writeResponse(w, "Failed")
			log.Errorf(ctx, "Bus Not Stored", err)
			return
		}
	default:
		writeResponse(w, "Operation Not Supported")
		log.Errorf(ctx, "Operation Not Supported")
		return
	}
	writeResponse(w, "Success")
}

func adminDriver(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u.Email != "bahus.vel@gmail.com" {
		writeResponse(w, "Unauthorized")
		return
	}

	operation := &DriverOperation{}
	if readRequest(r, operation) != nil{
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}

	dk := datastore.NewKey(ctx, "Driver", operation.TheDriver.Email, 0, nil)

	if operation.Operation == "upsert" {
		if _, err := datastore.Put(ctx, dk, &operation.TheDriver); err != nil{
			log.Errorf(ctx, "Failed to put in datastore %v", err)
		}
	} else if operation.Operation == "delete" {
		datastore.Delete(ctx, dk)
	}

	writeResponse(w, "Success")
}

func bussesAvailable(w http.ResponseWriter, r *http.Request){
	busses := []int{119, 235, 115, 113}
	res, _ := json.Marshal(busses)
	w.Write(res)
}

func driveBus(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)

	driveBus := &BusOperation{}
	if readRequest(r, driveBus) != nil{
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}

	btid, err := newOrGetBusTrip(ctx, driveBus.Bus.Number, "unitec")
	if err != nil {
		log.Errorf(ctx, "Saving Bus Trip Failed", err)
		writeResponse(w, "Failed")
		return
	}

	driver, uk := getDriver(ctx, u.Email)

	switch driveBus.Operation {
	case "drive":
		driver.CurrentBusTrip = btid
		driver.CurrentBus = driveBus.Bus.Number
	case "undrive":
		driver.CurrentBusTrip = ""
		driver.CurrentBus = 0
	}

	if _, err := datastore.Put(ctx, uk, driver); err != nil{
		log.Errorf(ctx, "Failed to put in datastore %v", err)
	}

	writeResponse(w, btid)

}

func logPosition(w http.ResponseWriter, r *http.Request){
	ctx := appengine.NewContext(r)
	u := user.Current(ctx)
	if u == nil {
		writeResponse(w, "Unauthorized")
		return
	}
	position := &Position{}
	if readRequest(r, position) != nil {
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}
	driver, _ := getDriver(ctx, u.Email)
	if driver.CurrentBusTrip == "" {
		writeResponse(w, "Not Driving Anything")
		return
	}
	err := storePosition(ctx, driver.CurrentBusTrip, driver.CurrentBus, position, "unitec")
	if err != nil {
		writeResponse(w, "Position Store Failed")
		return
	}
	writeResponse(w, "Success")
	log.Infof(ctx, "Position Store Successful, latitude: %f, longitude: %f", position.Latitude, position.Longitude)
}



func readRequest(r *http.Request, into interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, into); err  != nil {
		return err
	}
	return nil
}

func writeResponse(w http.ResponseWriter, message string){
	res, _ := json.Marshal(GenericResponse{Reponse:message})
	w.Write(res)
}

func driverExists(drivers []string, driver string) bool{
	for _, d := range drivers {
		if d == driver {
			return true
		}
	}
	return false
}

