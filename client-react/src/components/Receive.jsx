import React, { useEffect, useRef, useState } from "react";
import concatArrayBuffers from "./joinArrayBuffer";

const Receive = (props) => {

    const SERVER_WS = process.env.REACT_APP_SERVER_WS;

    const peerRef = useRef();
    const webSocketRef = useRef();

    var [linkName, setLinkName]=useState(<p>Name of file will appear here once it is downloaded</p>)
    var [fileName,setFileName]=useState("")
    var [fileType,setFileType]=useState("")
    var [fileSize,setFileSize]=useState(0)
    var [fileUploaded,setFileUploaded]=useState(false)

    useEffect(()=>{
        webSocketRef.current = new WebSocket(`${SERVER_WS}/recv?roomID=${props.match.params.roomID}`);

        webSocketRef.current.addEventListener("open", () => {
            webSocketRef.current.send(JSON.stringify({client:"receiver"}));
        });
    },[])

    useEffect(() => {
            webSocketRef.current.addEventListener("message", async (e) => {
                const message = JSON.parse(e.data);
                console.log(message)
				if (message.offer) {
                    handleOffer(message.offer);
                }
                if(message.filename){
                    setFileName(message.filename)
                } 
                if(message.filetype){
                    setFileType(message.filetype)
                } 
                if(message.filesize){
                    setFileSize(message.filesize)
                }

                if(message.fileuploaded){
                    console.log("ahem")
                    setFileUploaded(true)
                }

                if(!message.fileuploaded){
                    setFileUploaded(false)
                }

                if (message.answer) {
                    console.log("Receiving Answer");
                    peerRef.current.setRemoteDescription(
                        new RTCSessionDescription(message.answer)
                    );
                }

                if (message.iceCandidate) {
                    console.log("Receiving and Adding ICE Candidate");
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

    const handleOffer = async (offer) => {
        console.log("Received Offer, Creating Answer");
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
        console.log("Creating Peer Connection");
        const peer = new RTCPeerConnection({
            iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
        });

        peer.onnegotiationneeded = handleNegotiationNeeded;
        peer.onicecandidate = handleIceCandidateEvent;
        peer.ondatachannel = handleTrackEvent;

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
            webSocketRef.current.send(
                JSON.stringify({ iceCandidate: e.candidate })
            );
        }
    };

    const download =(e)=>{
        webSocketRef.current.send(
            JSON.stringify({request:"send"})
        );
    }
    const handleTrackEvent = async (e) => {
        
        var channel = e.channel;
        var data=new ArrayBuffer()
        channel.onmessage = async(event) =>{
            data=concatArrayBuffers(data,event.data)
            if(fileSize!==0&&fileName!==""&&fileType!==""&&data.byteLength===fileSize){
                var url=await arrayBufferToBase64(data,fileType,fileName)
                setLinkName(
                <div>
                    <h3>{fileName}</h3>
                    <a href={url} download={fileName}>
                        <button type="button" className="btn btn-primary btn-floating" >
                            <i className="fas fa-download"></i>
                        </button>
                    </a>
                </div>)
            }
        }        
    };


    var arrayBufferToBase64=async(Arraybuffer, Filetype, filename) =>{
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
        
        setLinkName(
            <div>
                <p>{filename}</p>
                <button type="button" className="btn btn-primary btn-floating">
                    <i className="fas fa-download" href={url} download={filename}></i>
                </button>
            </div>
            
        )
        return url
      }

    
    return (
        <div className="container m-sm-4 p-4 border col-sm-2">   
            <div className="row p-4"><h2>Download File</h2></div>
            <div className="row p-4">
                <button
                    className="btn btn-success"
                    disabled={fileUploaded}
                    onClick={download}
                    download
                >
                    Download
                </button>
            </div>
            <div className="row p-4">
                {linkName}
            </div>
        </div>
    );
};

export default Receive;