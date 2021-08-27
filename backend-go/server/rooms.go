package server

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//Participants are the sender and receivers
type Participant struct {
	Host bool
	Conn *websocket.Conn
}

//RoomMap has map with sender and receivers mapped to a roomID
type RoomMap struct {
	Mutex sync.RWMutex
	Map   map[string][]Participant
}

//Initializes the RoomMap struct
func (r *RoomMap) Init() {
	r.Map = make(map[string][]Participant)
}

//gets all the participants in the rooom
func (r *RoomMap) Get(roomID string) []Participant {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	return r.Map[roomID]
}

//generate unique ID and insert it in the hashmap then return it
func (r *RoomMap) CreateRoom() string {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, 8)

	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	roomID := string(b)
	r.Map[roomID] = []Participant{}
	return roomID
}

func (r *RoomMap) InsertIntoRoom(roomID string, host bool, conn *websocket.Conn) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	p := Participant{host, conn}

	log.Println("Inserting into room with RoomID: ", roomID)
	r.Map[roomID] = append(r.Map[roomID], p)
	log.Println(r)
}

func (r *RoomMap) DeleteFromRoom(roomID string, host bool, conn *websocket.Conn) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	var newmap []Participant

	for _, v := range r.Map[roomID] {
		if v.Conn == conn || !v.Host {
			newmap = append(newmap, v)
		}
	}
	r.Map[roomID] = newmap
}

func (r *RoomMap) DeleteThisFromRoom(roomID string, conn *websocket.Conn) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	var newmap []Participant

	for _, v := range r.Map[roomID] {
		if v.Conn != conn {
			newmap = append(newmap, v)
		}
	}
	r.Map[roomID] = newmap
}

func (r *RoomMap) DeleteRoom(roomID string) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()

	delete(r.Map, roomID)
}
