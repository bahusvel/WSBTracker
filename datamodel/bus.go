package datamodel

type Bus struct {
	number		int32
	drivers 	[]Driver
	children 	[]Child
}
