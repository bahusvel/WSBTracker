package httpserver

type Driver struct {
	Name 				string
	Email				string
	CurrentlyDriving	int32
	AllowedToDrive		[]int32
}