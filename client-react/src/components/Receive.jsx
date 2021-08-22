import React, { useEffect, useRef, useState } from "react";

const Receive = (props) => {
    let sentfile = useRef();
    const peerRef = useRef();
    const webSocketRef = useRef();

    var [linkEnabled,setLinkEnabled]=useState(false)
    var [linkName, setLinkName]=useState(<p>Name of file will appear here once it is downloaded</p>)

    useEffect(()=>{
        webSocketRef.current = new WebSocket(`ws://192.168.18.19:8000/recv?roomID=${props.match.params.roomID}`);

        webSocketRef.current.addEventListener("open", () => {
            webSocketRef.current.send(JSON.stringify({client:"receiver"}));
        });
    },[])

    useEffect(() => {
            webSocketRef.current.addEventListener("message", async (e) => {
                const message = JSON.parse(e.data);
                //console.log(message)

				if (message.offer) {
                    handleOffer(message.offer);
                }

                if (message.answer) {
                    console.log("Receiving Answer");
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
                        //console.log("Error Receiving ICE Candidate", err);
                    }
                }
            });
    });

    const handleOffer = async (offer) => {
        //console.log("Received Offer, Creating Answer");
        peerRef.current = createPeer();

        await peerRef.current.setRemoteDescription(
            new RTCSessionDescription(offer)
        );

        const answer = await peerRef.current.createAnswer();
        await peerRef.current.setLocalDescription(answer);

        webSocketRef.current.send(
            JSON.stringify({ answer: peerRef.current.localDescription })
        );
    };


    const createPeer = () => {
        //console.log("Creating Peer Connection");
        const peer = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });

        peer.onnegotiationneeded = handleNegotiationNeeded;
        peer.onicecandidate = handleIceCandidateEvent;
        peer.ondatachannel = handleTrackEvent;

        return peer;
    };

    const handleNegotiationNeeded = async () => {
        //console.log("Creating Offer");

        try {
            const myOffer = await peerRef.current.createOffer();
            await peerRef.current.setLocalDescription(myOffer);

            webSocketRef.current.send(
                JSON.stringify({ offer: peerRef.current.localDescription })
            );
        } catch (err) {}
    };

    const handleIceCandidateEvent = (e) => {
        //console.log("Found Ice Candidate");
        if (e.candidate) {
            //console.log(e.candidate);
            webSocketRef.current.send(
                JSON.stringify({ iceCandidate: e.candidate })
            );
        }
    };

    const download =(e)=>{
        //console.log("Requesting for download")
        webSocketRef.current.send(
            JSON.stringify({request:"send"})
        );
    }
    const handleTrackEvent = async (e) => {
        
        var channel = e.channel;
        const [name,type,size] = channel.label.split('....');
        const chunkSize = 16384;
        var data=new ArrayBuffer()
        channel.onmessage = async(event) =>{
            data=event.data
            console.log(data.byteLength)
            //console.log(Number(size))
            if(data.byteLength===Number(size)){
                var url=await arrayBufferToBase64(data,type,name)
                setLinkName(<a href={url} download={name}>{name}</a>)
                console.log(data)
            }
        }
        
        setLinkEnabled(true)
        
    };


    var arrayBufferToBase64=async(Arraybuffer, Filetype, fileName) =>{
        let binary = '';
        const bytes = new Uint8Array(Arraybuffer);
        const len = bytes.byteLength;
        for (let i = 0; i < len; i++) {
          binary += String.fromCharCode(bytes[i]);
        }
        const file = window.btoa(binary);
        const mimType = Filetype
        const url = `data:${mimType};base64,` + file;
        console.log(url)
        
        setLinkName(<a href={url} download={fileName}>{fileName}</a>)
        return url
      }

    
    return (
        <div>   
            <button
                className="btn btn-success"
                // disabled={!selectedFile}
                onClick={download}
                download
            >
                Download
            </button>
            {linkName}
        </div>
    );
};

export default Receive;