package server

import (
	"net/http"
)

//has a map of all the rooms
var AllRooms RoomMap

//Create a room and returns a ID and returns it
func SendFileRequestHandler(w http.ResponseWriter, f *http.Request) {
	w.Write([]byte("test"))
}

//Joins the client to the generated ID
func ReceiveFileRequestHandler(w http.ResponseWriter, f *http.Request) {
	w.Write([]byte("test2"))
}
