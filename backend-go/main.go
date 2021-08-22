package main

import (
	"backend-go/server"
	"log"
	"net/http"
)

func main() {

	server.AllRooms.Init()

	http.HandleFunc("/make", server.MakeLinkRequestHandler)
	http.HandleFunc("/recv", server.ReceiveFileRequestHandler)
	http.HandleFunc("/send", server.SendFileRequestHandler)

	log.Println("Starting server on port 8000")
	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatal(err)
	}
}
