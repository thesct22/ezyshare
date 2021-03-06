
import React, { useEffect, useRef, useState } from "react";

import UploadFiles from './FileUpload';


const Send = (props) => {

    const SERVER_WS = process.env.REACT_APP_SERVER_WS;

    const fileToSend = useRef();
    const peerRef = useRef();
    const webSocketRef = useRef();

    var [selectedFile, setSelectedFile] = useState(undefined);
    var [linkDisabled, setLinkDisabled] = useState(true);

    let addfile=async(val)=>{
        if (val===undefined){
            webSocketRef.current.send(
                JSON.stringify({fileuploaded:false})
            );
        }
        else{
            webSocketRef.current.send(
                JSON.stringify({fileuploaded:true})
            );
        }
      setSelectedFile(val)
    }

    useEffect(()=>{
        webSocketRef.current = new WebSocket(`${SERVER_WS}/send?roomID=${props.match.params.roomID}`);
        webSocketRef.current.addEventListener("open", () => {
            webSocketRef.current.send(JSON.stringify({ client: "sender" }));
        });
    },[]);

    useEffect(() => {
      
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
              peerRef.current.setRemoteDescription(
                  new RTCSessionDescription(message.answer)
              );
          }

          if (message.iceCandidate) {
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

    const callUser = () => {
        peerRef.current = createPeer();
        if(fileToSend.current!==undefined){
          console.log("sentfile")
          webSocketRef.current.send(
            JSON.stringify({filename:fileToSend.current.name,filetype:fileToSend.current.type,filesize:fileToSend.current.size})
        );

          var labelToSend=fileToSend.current.name
          const chunkSize = 16384;
          var channel=peerRef.current.createDataChannel(labelToSend,{maxRetransmits:4,});
          channel.onopen = function(event) {

            var arrayBuffer;
            var fileReader = new FileReader();
            fileReader.onload = async function(event2) {
                arrayBuffer = event2.target.result;
                var begin=0
                var end=chunkSize
                while (true){
                  channel.send(arrayBuffer.slice(begin,end))
                  begin+=chunkSize
                  end+=chunkSize
                  if(begin>fileToSend.current.size)
                    break
                }
                await channel.send(arrayBuffer);
            };
            fileReader.readAsArrayBuffer(fileToSend.current);
          }
        }
        else console.log("User not joined")
    };

    const createPeer = () => {
        console.log("creating peer")
        const peer = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });

        peer.onnegotiationneeded = handleNegotiationNeeded;

        return peer;
    };

    const handleNegotiationNeeded = async () => {

        try {
            const myOffer = await peerRef.current.createOffer();
            await peerRef.current.setLocalDescription(myOffer);

            webSocketRef.current.send(
                JSON.stringify({ offer: peerRef.current.localDescription })
            );
        } catch (err) {}
    };

    return(
      <div>
        <UploadFiles addfile={addfile} disabled={linkDisabled} />
      </div>
    );
};

export default Send;