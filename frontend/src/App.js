import logo from "./logo.svg";
import "./App.css";
import { useState, useEffect } from "react";

function App() {
  const [pcSender, setpcSender] = useState(
    new RTCPeerConnection({
      iceServers: [
        {
          urls: "stun:stun.l.google.com:19302",
        },
      ],
    })
  );
  const [pcReciever, setpcReciever] = useState(
    new RTCPeerConnection({
      iceServers: [
        {
          urls: "stun:stun.l.google.com:19302",
        },
      ],
    })
  );
  const [meetingId, setmeetingId] = useState();
  const [peerId, setpeerId] = useState();
  const [userId, setuserId] = useState();
  const searchParams = new URLSearchParams(document.location.search);

  const startCall = () => {
    // sender part of the call
    navigator.mediaDevices
      .getUserMedia({ video: true, audio: true })
      .then((stream) => {
        var senderVideo = document.getElementById("senderVideo");
        senderVideo.srcObject = stream;
        var tracks = stream.getTracks();
        for (var i = 0; i < tracks.length; i++) {
          pcSender.addTrack(stream.getTracks()[i]);
        }
        pcSender
          .createOffer()
          .then((d) => pcSender.setLocalDescription(d));
      });
    // you can use event listner so that you inform he is connected!
    pcSender.addEventListener("connectionstatechange", (event) => {
      if (pcSender.connectionState === "connected") {
        console.log("horray!");
      }
    });

    // receiver part of the call
    pcReciever.addTransceiver("video", { direction: "recvonly" });

    pcReciever
      .createOffer()
      .then((d) => pcReciever.setLocalDescription(d));

    pcReciever.ontrack = function (event) {
      var receiverVideo = document.getElementById("receiverVideo");
      receiverVideo.srcObject = event.streams[0];
      receiverVideo.autoplay = true;
      receiverVideo.controls = true;
    };
  };

  useEffect(() => {
    console.log(searchParams);
    setmeetingId(searchParams.get("meetingId"));
    setpeerId(searchParams.get("peerId"));
    setuserId(searchParams.get("userId"));
    pcSender.onicecandidate = (event) => {
      if (event.candidate === null) {
        const localDescription = pcSender.localDescription.toJSON();
        const sdpPayload = btoa(JSON.stringify(localDescription));
        const url = `http://localhost:8080/webrtc/sdp/m/${meetingId}/c/${userId}/p/${peerId}/s/true`;
        const body = { sdp: sdpPayload };
        fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        })
          .then((response) => response.json())
          .then((response) => {
            const remoteDescription = JSON.parse(atob(response.Sdp));
            pcSender.setRemoteDescription(
              new RTCSessionDescription(remoteDescription)
            );
          })
          .catch((error) => {
            console.error("Error:", error);
          });
      }
    };
    pcReciever.onicecandidate = (event) => {
      if (event.candidate === null) {
        const localDescription = pcReciever.localDescription.toJSON();
        const sdpPayload = btoa(JSON.stringify(localDescription));
        const url = `http://localhost:8080/webrtc/sdp/m/${meetingId}/c/${userId}/p/${peerId}/s/false`;
        const body = { sdp: sdpPayload };

        fetch(url, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        })
          .then((response) => response.json())
          .then((response) => {
            const remoteDescription = JSON.parse(atob(response.Sdp));
            pcReciever.setRemoteDescription(
              new RTCSessionDescription(remoteDescription)
            );
          })
          .catch((error) => {
            console.error("Error:", error);
          });
      }
    };
  }, []);
  return (
    <div className="App">
      <header className="App-header">
        <button onClick={startCall} className="start-call">
          Start the call!
        </button>
        <div className="container_row">
          <video
            autoPlay
            id="senderVideo"
            width="500"
            height="500"
            controls
            muted
          ></video>
          <div className="layer2">
            <video
              autoPlay
              id="receiverVideo"
              width="160"
              height="120"
              controls
              muted
            ></video>
          </div>
        </div>
      </header>
    </div>
  );
}

export default App;
