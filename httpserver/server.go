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
	"golang.org/x/net/context"
	"strconv"
	"time"
)

type BusTrip struct {
	BusNumber		int32
	Drivers 		[]string
	LocationTrace	[]Position
}

type Position struct {
	Latitude	float64
	Longitude	float64
}

type GenericResponse struct {
	Reponse		string
}

type DriveBus struct {
	BusNumber	int32
	Drive		bool
}

type DriverOperation struct{
	TheDriver 		Driver
	Operation 		string
}


func init() {
    http.HandleFunc("/", home)
	http.HandleFunc("/position/log", logPosition)
	http.HandleFunc("/busses/available", bussesAvailable)
	http.HandleFunc("/busses/drive", driveBus)
	http.HandleFunc("/admin/driver", adminDriver)
	http.HandleFunc("/logout", logout)
}


func home(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Hello, world!")
}

func logout(w http.ResponseWriter, r *http.Request) {
	url, _ := user.LogoutURL(appengine.NewContext(r), "/")
	//w.Write([]byte("<a href=\"" + url + "\">Logout</a>"))
	http.Redirect(w, r, url, 301)
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

	driveBus := &DriveBus{}
	if readRequest(r, driveBus) != nil{
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}

	driver, uk := getDriver(ctx, u.Email)

	if driveBus.Drive {
		driver.CurrentlyDriving = driveBus.BusNumber
	} else {
		driver.CurrentlyDriving = 0
	}
	if _, err := datastore.Put(ctx, uk, driver); err != nil{
		log.Errorf(ctx, "Failed to put in datastore %v", err)
	}

	trip, btk := getBusTrip(ctx, driveBus.BusNumber)
	if trip == nil {
		trip = new(BusTrip)
		log.Infof(ctx, "Trip Does Not Exist")
		trip.BusNumber = driveBus.BusNumber
		trip.Drivers = []string{driver.Email}
	} else {
		if !driverExists(trip.Drivers, driver.Email) {
			trip.Drivers = append(trip.Drivers, driver.Email)
		}
	}
	if _, err := datastore.Put(ctx, btk, trip); err != nil{
		log.Errorf(ctx, "Failed to put in datastore %v", err)
	}
	writeResponse(w, "Success")

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
	if driver.CurrentlyDriving == 0 {
		writeResponse(w, "Unauthorized")
		return
	}
	_, btk := getBusTrip(ctx, driver.CurrentlyDriving)
	ctime := int(time.Now().Unix())
	k := datastore.NewKey(ctx, "Position", strconv.Itoa(ctime), 0, btk)
	if _, err := datastore.Put(ctx, k, position); err != nil {
		log.Errorf(ctx, "Failed to put in datastore %v", err)
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

func getDriver(ctx context.Context, email string) (*Driver, *datastore.Key){
	uk := datastore.NewKey(ctx, "Driver", email, 0, nil)

	driver := new(Driver)
	err := datastore.Get(ctx, uk, driver)
	if err != nil || driver.Email != email{
		return nil, uk
	}
	return driver, uk
}

func getBusTrip(ctx context.Context, busNumber int32) (*BusTrip, *datastore.Key) {
	btk := datastore.NewKey(ctx, "BusTrip", "", int64(busNumber), nil)
	trip := new(BusTrip)
	if datastore.Get(ctx, btk, trip) != nil {
		return nil, btk
	}
	return trip, btk
}

func driverExists(drivers []string, driver string) bool{
	for _, d := range drivers {
		if d == driver {
			return true
		}
	}
	return false
}

