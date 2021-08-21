import React from 'react'
import {useEffect,useRef} from 'react';

const Room=(props)=>{
    useEffect(()=>{
        console.log(props)
        const ws = new WebSocket(`ws://192.168.18.19:8000/recv?roomID=${props.match.params.roomID}`);
        ws.addEventListener("open",()=>{
            console.log("sending")
            ws.send(JSON.stringify({join:"true"}));
        });

        ws.addEventListener("message",(e)=>{
            console.log(e.data)
        })
    })
    return(
        <div>
            <video autoPlay controls={true}></video>
            <video autoPlay controls={true}></video>
        </div>
    )
}

export default Room;