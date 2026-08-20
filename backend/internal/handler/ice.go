package handler

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type ICEResponse struct {
	ICEServers []ICEServer `json:"iceServers"`
}

func HandleICEServers(w http.ResponseWriter, r *http.Request) {
	turnServerURL := os.Getenv("TURN_SERVER_URL")
	turnSecret := os.Getenv("TURN_SECRET")

	iceServers := []ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
		},
	}

	if turnServerURL != "" && turnSecret != "" {
		// Generate ephemeral Coturn HMAC-SHA1 credentials (valid for 24 hours)
		ttl := 24 * time.Hour
		timestamp := time.Now().Add(ttl).Unix()
		username := fmt.Sprintf("%d:ezyshare-user", timestamp)

		mac := hmac.New(sha1.New, []byte(turnSecret))
		mac.Write([]byte(username))
		credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

		iceServers = append(iceServers, ICEServer{
			URLs:       []string{turnServerURL},
			Username:   username,
			Credential: credential,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ICEResponse{ICEServers: iceServers})
}
