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
	"crypto/rand"
	"golang.org/x/net/context"
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
	Token 	string
}

type AuthenticationRequest struct {
	Email 		string
	Password 	string
}

type PositionRequest struct {
	Position
	Token string
}

func init() {
    http.HandleFunc("/", home)
	http.HandleFunc("/login", login)
	http.HandleFunc("/position/log", logPosition)
	http.HandleFunc("/busses/available", bussesAvailable)
	http.HandleFunc("/busses/drive", driveBus)
	http.HandleFunc("/busses/location", busLocation)
	http.HandleFunc("/admin/driver", adminDriver)
	http.HandleFunc("/admin/bus", adminBus)
	http.HandleFunc("/position/test", positionTest)
	http.HandleFunc("/logout", logout)
}


func home(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "It Works!")
}

func login(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	request := &AuthenticationRequest{}
	if readRequest(r, request) != nil {
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}
	driver, key := getDriver(ctx, request.Email)
	if driver == nil {
		writeResponse(w, "Unauthorized")
		return;
	}
	if driver.Email == request.Email && driver.Password == request.Password{
		token := generateToken()
		driver.Token = token
		if _, err := datastore.Put(ctx, key, driver); err != nil{
			log.Errorf(ctx, "Failed to put in datastore %v", err)
		}
		writeResponse(w, token)
		return;
	}
	writeResponse(w, "Unauthorized")
}

func positionTest(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	position := &PositionRequest{}
	if err := readRequest(r, position); err != nil {
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received", err)
		return
	}
    uEmail := getUserEmail(ctx, position.Token)
	if uEmail == "" {
		writeResponse(w, "Unauthorized")
		return
	}

	bytes, _ := json.Marshal(position.Position);
	w.Write(bytes)
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
	uEmail := getUserEmail(ctx, query.Get("token"))
	if uEmail == "" {
		writeResponse(w, "Unauthorized")
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

	uEmail := getUserEmail(ctx, busOp.Token)
	if uEmail == "" {
		writeResponse(w, "Unauthorized")
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
	driveBus := &BusOperation{}
	if readRequest(r, driveBus) != nil{
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}

	uEmail := getUserEmail(ctx, driveBus.Token)
	if uEmail == "" {
		writeResponse(w, "Unauthorized")
		return
	}

	btid, err := newOrGetBusTrip(ctx, driveBus.Bus.Number, "unitec")
	if err != nil {
		log.Errorf(ctx, "Saving Bus Trip Failed", err)
		writeResponse(w, "Failed")
		return
	}

	driver, uk := getDriver(ctx, uEmail)

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
	position := &PositionRequest{}
	if readRequest(r, position) != nil {
		writeResponse(w, "Unreadable Request")
		log.Errorf(ctx, "Unredable request received")
		return
	}
	uEmail := getUserEmail(ctx, position.Token)
	if uEmail == "" {
		writeResponse(w, "Unauthorized")
		return
	}

	driver, _ := getDriver(ctx, uEmail)
	if driver.CurrentBusTrip == "" {
		writeResponse(w, "Not Driving Anything")
		return
	}
	err := storePosition(ctx, driver.CurrentBusTrip, driver.CurrentBus, &position.Position, "unitec")
	if err != nil {
		writeResponse(w, "Position Store Failed")
		return
	}
	writeResponse(w, "Success")
	log.Infof(ctx, "Position Store Successful, latitude: %f, longitude: %f", position.Latitude, position.Longitude)
}



func readRequest(r *http.Request, into interface{}) error {
	reader := r.Body
	body, err := ioutil.ReadAll(reader)
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

func getUserEmail(ctx context.Context, token string) string{
	log.Infof(ctx, "Token", token)
	driver := getDriverByToken(ctx, token)
	if driver == nil{
		return ""
	}
	return driver.Email
}

func driverExists(drivers []string, driver string) bool{
	for _, d := range drivers {
		if d == driver {
			return true
		}
	}
	return false
}

func generateToken() string{
    const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, 32)
    rand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

