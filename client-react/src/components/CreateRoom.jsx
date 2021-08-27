import React, { useState } from 'react';
import QRCode from "react-qr-code";


const CreateRoom =(props)=>{

    var makeid=()=> {
        var result           = '';
        var characters       = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
        var charactersLength = characters.length;
        for ( var i = 0; i < 8; i++ ) {
          result += characters.charAt(Math.floor(Math.random() * 
     charactersLength));
       }
       console.log(result)
       return result;
    }
    var [roomID,setRoomID]=useState("");

    const customlink=(event)=>{
        //console.log(event)
        setRoomID(event.target.value)
    }
    const toSend=async (e)=>{
        e.preventDefault()
        props.history.push(`/send/${roomID}`)
    }
    const linkToRecv=()=>{
        console.log(roomID)
        return "http://192.168.18.19/recv/"+roomID
    }

    const linkToSend=()=>{
        console.log(roomID)
        return "http://192.168.18.19/send/"+roomID
    }    

    const makeLink=()=>{
        var roomid=makeid();
        setRoomID(roomid);
    }

    const toRecv=(e)=>{
        e.preventDefault()
        props.history.push(`/recv/${roomID}`)
    }

    return(
        <div className="container">
            <div className="row">
                <div className="col">
                    <h2>Create Link</h2>
                </div>
            </div>
            <div className="row">
                <div className="col">
                    <input type="text" name="roomName" value={roomID} onChange={customlink}></input>
                </div>
                <div className="col">
                    <button onClick={makeLink} className="btn btn-default">Create</button>
                </div>
            </div>
            <div className="row">
                <div className="col">
                    <h2>Send Files</h2>
                </div>
            </div>
            <div className="row">
                <div className="col">
                    <input type="text" name="sendLink" value={roomID===""?"":"192.168.18.19/send/"+roomID} readOnly></input>
                </div>
                <div className="col">
                    <button onClick={toSend} className="btn btn-default">Send</button>
                </div>
            </div>
            <div className="row">
                <div className="col">
                    <h2>Receive Files</h2>
                </div>
            </div>
            <div className="row">
                <div className="col">
                    <input type="text" name="recvLink" value={roomID===""?"":"192.168.18.19/recv/"+roomID} readOnly ></input>
                </div>
                <div className="col">
                    <button onClick={toRecv} className="btn btn-default">Receive</button>
                </div>
            </div>
            <div className="row">
                <div className="col-sm-3 m-2 p-2 bg-light border">
                <div className="card">
                    <QRCode value={roomID===""?"":"192.168.18.19/send/"+roomID} className="card-img-top m-1 my-2"/>
                    <div className="card-body">
                        <h5 className="card-title">Send</h5>
                        <p className="card-text">
                            Scan this QR code on your phone to send files.
                        </p>
                    </div>
                </div>
                </div>
                <div className="col-sm-3 m-2 p-2 bg-light border">
                <div className="card">
                    <QRCode value={roomID===""?"":"192.168.18.19/recv/"+roomID} className="card-img-top m-1 my-2" />
                    <div className="card-body">
                        <h5 className="card-title">Receive</h5>
                        <p className="card-text">
                            Scan this QR code on your phone to receive files.
                        </p>
                    </div>
                </div>
                </div>
            </div>

        </div>
    );
}

export default CreateRoom