package server

import (
	"encoding/json"
	//"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

//has a map of all the rooms
var AllRooms RoomMap

//Create a room and returns a ID and returns it
func MakeLinkRequestHandler(w http.ResponseWriter, f *http.Request) {
	roomID := AllRooms.CreateRoom()
	type resp struct {
		RoomID string `json:"room_id"`
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp{RoomID: roomID})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type broadcastMessage struct {
	Message map[string]interface{}
	RoomID  string
	Client  *websocket.Conn
}

var broadcast = make(chan broadcastMessage)

func broadcaster() {
	for {
		msg := <-broadcast
		for _, client := range AllRooms.Map[msg.RoomID] {
			if client.Conn != msg.Client {
				//fmt.Println(msg)
				err := client.Conn.WriteJSON(msg.Message)
				// if err == nil {
				// 	//fmt.Println(msg)

				// }
				if err != nil {
					AllRooms.DeleteThisFromRoom(msg.RoomID, client.Conn)
				}
			}
			if !client.Host && client.Conn != msg.Client {
				err := client.Conn.WriteJSON("added-recv")
				//fmt.Println(client.Conn.RemoteAddr())
				if err != nil {
					AllRooms.DeleteThisFromRoom(msg.RoomID, client.Conn)
				}
			}
		}
	}
}

//Joins the client to the generated ID
func ReceiveFileRequestHandler(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomID")
	log.Println(roomID)
	if roomID == "" {
		log.Println("roomID missing in URL Parameter")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Websocket Upgrade Error: ", err)
	}
	defer ws.Close()
	AllRooms.InsertIntoRoom(roomID, false, ws)

	go broadcaster()

	for {
		var msg broadcastMessage
		err := ws.ReadJSON((&msg.Message))
		if err != nil {

			//fmt.Println("Hello there")
			//log.Println(err)
			break
		}

		if err == nil {
			msg.Client = ws
			msg.RoomID = roomID

			//log.Println(msg.Message)

			broadcast <- msg
		}
	}
}

func SendFileRequestHandler(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("roomID")
	//log.Println(roomID)
	if roomID == "" {
		log.Println("roomID missing in URL Parameter")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("Websocket Upgrade Error: ", err)
	}
	//defer ws.Close()
	AllRooms.InsertIntoRoom(roomID, true, ws)
	AllRooms.DeleteFromRoom(roomID, true, ws)

	go broadcaster()

	for {
		var msg broadcastMessage
		err := ws.ReadJSON((&msg.Message))
		if err != nil {

			//ws.Close()
			//fmt.Println("Hello there Kenobi")
			break
			//fmt.Println(err)
			//log.Fatal(err)
		} else {
			msg.Client = ws
			msg.RoomID = roomID

			//fmt.Println("Hello there")
			//log.Println(msg.Message)

			broadcast <- msg
		}
	}
}
