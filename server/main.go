package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v2"
	cors "github.com/rs/cors/wrapper/gin"
	signal "github.com/kr-2003/go-react-webrtc/signal"
)

const (
	rtcpPLIInterval = time.Second * 3
)

type Sdp struct {
	Sdp string
}

func main() {
	file, err := os.OpenFile("info.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	log.SetOutput(file)
	router := gin.Default()

	router.Use(cors.Default())

	peerConnectionMap := make(map[string]chan *webrtc.Track)

	m := webrtc.MediaEngine{}

	m.RegisterCodec(webrtc.NewRTPVP8Codec(webrtc.DefaultPayloadTypeVP8, 90000))

	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))

	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	router.POST("/webrtc/sdp/m/:meetingId/c/:userId/p/:peerId/s/:isSender", func(c *gin.Context) {
		isSender, _ := strconv.ParseBool(c.Param("isSender"))
		userId := c.Param("userId")
		peerId := c.Param("peerId")

		var session Sdp

		if err := c.ShouldBindJSON(&session); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		offer := webrtc.SessionDescription{}

		signal.Decode(session.Sdp, &offer)

		peerConnection, err := api.NewPeerConnection(peerConnectionConfig)

		if err != nil {
			log.Fatal(err)
		}
		if !isSender {
			recieveTrack(peerConnection, peerConnectionMap, peerId)
		} else {
			createTrack(peerConnection, peerConnectionMap, userId)
		}

		peerConnection.SetRemoteDescription(offer)

		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Fatal(err)
		}

		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			log.Fatal(err)
		}
		c.JSON(http.StatusOK, Sdp{Sdp: signal.Encode(answer)})
	})
	router.Run(":8080")
}

func recieveTrack(peerConnection *webrtc.PeerConnection, peerConnectionMap map[string]chan *webrtc.Track, peerId string) {
	if _, ok := peerConnectionMap[peerId]; !ok {
		peerConnectionMap[peerId] = make(chan *webrtc.Track, 1)
	}
	localTrack := <-peerConnectionMap[peerId]
	peerConnection.AddTrack(localTrack)
}

func createTrack(peerConnection *webrtc.PeerConnection, peerConnectionMap map[string]chan *webrtc.Track, currentUserID string) {
	if _, err := peerConnection.AddTransceiver(webrtc.RTPCodecTypeVideo); err != nil {
		log.Fatal(err)
	}

	peerConnection.OnTrack(func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(rtcpPLIInterval)
			for range ticker.C {
				if rtcpSendErr := peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remoteTrack.SSRC()}}); rtcpSendErr != nil {
					fmt.Println(rtcpSendErr)
				}
			}
		}()

		localTrack, newTrackErr := peerConnection.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", "pion")
		if newTrackErr != nil {
			log.Fatal(newTrackErr)
		}

		localTrackChan := make(chan *webrtc.Track, 1)
		localTrackChan <- localTrack
		if existingChan, ok := peerConnectionMap[currentUserID]; ok {
			existingChan <- localTrack
		} else {
			peerConnectionMap[currentUserID] = localTrackChan
		}

		rtpBuf := make([]byte, 1400)
		for {
			i, readErr := remoteTrack.Read(rtpBuf)
			if readErr != nil {
				log.Fatal(readErr)
			}

			if _, err := localTrack.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
				log.Fatal(err)
			}
		}
	})
}
