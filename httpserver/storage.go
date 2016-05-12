package httpserver
import (
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	uuid "github.com/nu7hatch/gouuid"
	"time"
	"strconv"
	"google.golang.org/appengine/log"
)


type Admin struct {
	Name	string
	Email	string
}

type Bus struct {
	Number		int
	Drivers 	[]string
	CurrentTrip	string
}

type Position struct {
	Time		int
	Latitude	float64
	Longitude	float64
}

type Child struct {
	name			string
	parentIDs		[]string
}

type Organization struct {
	ID		string
	Admins	[]Admin
	Busses	[]Bus
}

type Driver struct {
	Email          string
	Password	   string
	Name           string
	CurrentBusTrip string
	CurrentBus		int
	AllowedToDrive []int
}

type BusTrip struct {
	ID				string
	BusNumber		int
	Drivers 		[]string
	Children		[]Child
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

func newOrGetBusTrip(ctx context.Context, busNumber int, organizationid string) (string, error){
	bus, bk, err := getBus(ctx, busNumber, organizationid)
	if err != nil {
		return "", err
	}
	if bus.CurrentTrip != "" {
		return bus.CurrentTrip, nil
	}
	btid := uuidGen()
	bt := &BusTrip{ID:btid, BusNumber:busNumber}
	btk := datastore.NewKey(ctx, "BusTrip", btid, 0, bk)
	_, err = datastore.Put(ctx, btk, bt)
	if err != nil {
		return "", err
	}
	bus.CurrentTrip = btid
	err = storeBus(ctx, bus, organizationid)
	return btid, err
}

func storeBus(ctx context.Context, bus *Bus, organizationid string) error{
	ok := datastore.NewKey(ctx, "Organization", organizationid, 0, nil)
	bk := datastore.NewKey(ctx, "Bus", "", int64(bus.Number), ok)
	_, err := datastore.Put(ctx, bk, bus)
	return err
}

func getBus(ctx context.Context, busNumber int, organizationid string) (*Bus, *datastore.Key, error){
	ok := datastore.NewKey(ctx, "Organization", organizationid, 0, nil)
	bk := datastore.NewKey(ctx, "Bus", "", int64(busNumber), ok)
	var bus Bus
	err := datastore.Get(ctx, bk, &bus)
	return &bus, bk, err
}

func storePosition(ctx context.Context, btid string, busNumber int,position *Position, organizationid string) error {
	ctime := int(time.Now().Unix())
	position.Time = ctime
	_, btk := getBusTrip(ctx, btid, busNumber, organizationid)
	k := datastore.NewKey(ctx, "Position", strconv.Itoa(ctime), 0, btk)
	if _, err := datastore.Put(ctx, k, position); err != nil {
		log.Errorf(ctx, "Failed to put in datastore %v", err)
		return err
	}
	return nil
}

func getBusTrip(ctx context.Context, btid string, busNumber int, organizationid string) (*BusTrip, *datastore.Key) {
	ok := datastore.NewKey(ctx, "Organization", organizationid, 0, nil)
	bk := datastore.NewKey(ctx, "Bus", "", int64(busNumber), ok)
	btk := datastore.NewKey(ctx, "BusTrip", btid, 0, bk)
	trip := new(BusTrip)
	if datastore.Get(ctx, btk, trip) != nil {
		return nil, btk
	}
	return trip, btk
}

func uuidGen() string{
	id, _  := uuid.NewV4()
	return id.String()
}