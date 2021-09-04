package main

import (
	"backend-go/server"
	"log"
	"net/http"
	"os"
)

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "6969"
		//log.Fatal("$PORT must be set")
	}

	server.AllRooms.Init()
	http.HandleFunc("/", server.MakeLinkRequestHandler)
	http.HandleFunc("/make", server.MakeLinkRequestHandler)
	http.HandleFunc("/recv", server.ReceiveFileRequestHandler)
	http.HandleFunc("/send", server.SendFileRequestHandler)

	//log.Println("Starting server on given port")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
