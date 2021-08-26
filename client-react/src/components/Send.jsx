
import React, { useEffect, useRef, useState } from "react";

import UploadFiles from './FileUpload';


const Send = (props) => {
    const fileToSend = useRef();
    const peerRef = useRef();
    const webSocketRef = useRef();

    var [selectedFile, setSelectedFile] = useState(undefined);
    var [linkDisabled, setLinkDisabled] = useState(true);

    let addfile=async(val)=>{
      setSelectedFile(val)
    }

    // useEffect(()=>{
    //   webSocketRef.current = new WebSocket(`ws://192.168.18.19:8000/send?roomID=${props.match.params.roomID}`);

    //   webSocketRef.current.addEventListener("open", () => {
    //       webSocketRef.current.send(JSON.stringify({ client: "sender" }));
    //   });
    // },[]);

    useEffect(() => {
      webSocketRef.current = new WebSocket(`ws://192.168.18.19:8000/send?roomID=${props.match.params.roomID}`);

      webSocketRef.current.addEventListener("open", () => {
          webSocketRef.current.send(JSON.stringify({ client: "sender" }));
      });
      if(selectedFile!==undefined)
        fileToSend.current= selectedFile;
      webSocketRef.current.addEventListener("message", async (e) => {
          const message = JSON.parse(e.data);

          if (message.client==="receiver") {
              console.log("Receiver joined");
              setLinkDisabled(false)

          }
          
          if (message.request==="send") {
            callUser();
          }

          if (message.answer) {
              //console.log("Receiving Answer");
              peerRef.current.setRemoteDescription(
                  new RTCSessionDescription(message.answer)
              );
          }

          if (message.iceCandidate) {
              //console.log("Receiving and Adding ICE Candidate");
              try {
                  await peerRef.current.addIceCandidate(
                      message.iceCandidate
                  );
              } catch (err) {
                  console.log("Error Receiving ICE Candidate", err);
              }
          }
      });
    });

    // useEffect(()=>{
    //   if (selectedFile!==undefined){
    //     //console.log("This file is added")
    //     //console.log(selectedFile)}
    // })

    const callUser = () => {
        console.log("Calling Other User");
        peerRef.current = createPeer();
        if(fileToSend.current!==undefined){
          console.log("sentfile")
          webSocketRef.current.send(
            JSON.stringify({filename:fileToSend.current.name,filetype:fileToSend.current.type,filesize:fileToSend.current.size})
        );
          //console.log(fileToSend)

          var labelToSend=fileToSend.current.name
          const chunkSize = 16384;
          var channel=peerRef.current.createDataChannel(labelToSend,{maxRetransmits:4,});
          channel.onopen = function(event) {

            console.log(fileToSend.current)
            var arrayBuffer;
            var fileReader = new FileReader();
            fileReader.onload = async function(event2) {
                arrayBuffer = event2.target.result;
                var begin=0
                var end=chunkSize
                while (true){
                  console.log(arrayBuffer.slice(begin,end))
                  channel.send(arrayBuffer.slice(begin,end))
                  begin+=chunkSize
                  end+=chunkSize
                  if(begin>fileToSend.current.size)
                    break
                }
                //console.log(arrayBuffer);
                await channel.send(arrayBuffer);
            };
            fileReader.readAsArrayBuffer(fileToSend.current);
            //channel.send(arrayBuffer);
          }
        }
        else console.log("User not joined")
    };

    const createPeer = () => {
        console.log("Creating Peer Connection");
        const peer = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });

        peer.onnegotiationneeded = handleNegotiationNeeded;
        peer.onicecandidate = handleIceCandidateEvent;

        return peer;
    };

    const handleNegotiationNeeded = async () => {
        console.log("Creating Offer");

        try {
            const myOffer = await peerRef.current.createOffer();
            await peerRef.current.setLocalDescription(myOffer);

            webSocketRef.current.send(
                JSON.stringify({ offer: peerRef.current.localDescription })
            );
        } catch (err) {}
    };

    const handleIceCandidateEvent = (e) => {
        console.log("Found Ice Candidate");
        if (e.candidate) {
            //console.log(e.candidate);
            webSocketRef.current.send(
                JSON.stringify({ iceCandidate: e.candidate })
            );
        }
    };

    return(
      <div>
        <UploadFiles addfile={addfile} disabled={linkDisabled} />
      </div>
    );
};

export default Send;