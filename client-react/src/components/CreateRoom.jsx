import React, { useState } from 'react';
import QRCode from "react-qr-code";

import { ToastContainer, toast } from 'react-toastify';
import 'react-toastify/dist/ReactToastify.css';


const CreateRoom =(props)=>{
    
    const SERVER_URL = process.env.REACT_APP_SERVER_URL;
    const baseUrl = window.location.href;
    console.log(SERVER_URL)



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

    const makeLink=()=>{
        var roomid=makeid();
        setRoomID(roomid);
    }

    const toRecv=(e)=>{
        e.preventDefault()
        props.history.push(`/recv/${roomID}`)
    }

    return(
        <div className="container m-3">
            <ToastContainer 
                autoClose={1500}
                hideProgressBar
                closeOnClick
                draggable
            />
            <div className=" col-12 col-sm-8 border border-2 rounded">
            <div className="mt-3 m-sm-5 border p-4 p-sm-5">
                <div className="row">
                    <div className="col">
                        <h2>Create Link</h2>
                    </div>
                </div>
                <div className="row">
                    <div className="col-12 col-sm-4">
                        <div className="form-outline">
                            <input type="text" id="form1" className="form-control" value={roomID} onChange={customlink}/>
                            <label className="form-label" htmlFor="form1">Unique ID</label>
                        </div>
                    </div>
                    <div className="col-sm-4"></div>
                    <div className="col-sm-2 d-flex justify-content-top">
                        <button onClick={makeLink} className="btn btn-success align-self-start">Create</button>
                    </div>
                </div>
            </div>
            <div className="mt-3 m-sm-5 border p-4 p-sm-5">
                <div className="row">
                    <div className="col">
                        <h2>Send Files</h2>
                    </div>
                </div>
                <div className="row">
                    <div className="col-12 col-sm-8">
                        <div className="form-outline">
                            <input type="text" 
                                id="form2" 
                                className="form-control" 
                                value={roomID===""?"":`${baseUrl}send/`+roomID} 
                                onClick={() =>{
                                        navigator.clipboard.writeText(roomID===""?"":`${baseUrl}send/`+roomID);
                                        toast("Copied to Clipboard!");
                                    }
                                }
                                readOnly/>
                            <label className="form-label" htmlFor="form2">Click the above text box to copy</label>
                        </div>
                    </div>
                    <div className="col-sm-2 d-flex justify-content-top">
                        <button onClick={toSend} className="btn btn-info align-self-start">Send</button>
                    </div>
                </div>
            </div>
            <div className="mt-3 m-sm-5 border p-4 p-sm-5">
                <div className="row">
                    <div className="col">
                        <h2>Receive Files</h2>
                    </div>
                </div>
                <div className="row">
                    <div className="col-12 col-sm-8">
                        <div className="form-outline">
                            <input type="text" 
                                id="form3" 
                                className="form-control" 
                                value={roomID===""?"":`${baseUrl}recv/`+roomID}
                                onClick={() =>{
                                        navigator.clipboard.writeText(roomID===""?"":`${baseUrl}recv/`+roomID);
                                        toast("Copied to Clipboard!");
                                    }
                                }
                                readOnly />
                            <label className="form-label" htmlFor="form3">Click the above text box to copy</label>
                        </div>
                    </div>
                    <div className="col-sm-2 d-flex justify-content-top">
                        <button onClick={toRecv} className="btn btn-warning align-self-start">Receive</button>
                    </div>
                </div>
            </div>
            <div className="row row-cols-1 g-4">
                <div className="col-10 col-sm-5 m-2 p-2">
                    <div className="card text-center shadow-5 d-flex justify-content-center">
                        <div className="d-flex justify-content-center">
                            <QRCode value={roomID===""?"":`${baseUrl}send/`+roomID} className="card-img-top m-5"/>
                        </div>
                        <div className="card-body">
                            <h5 className="card-title">Send</h5>
                            <p className="card-text">
                                Scan this QR code on your phone to send files.
                            </p>
                        </div>
                    </div>
                </div>
                <div className="col-10 col-sm-5 m-2 p-2">
                    <div className="card text-center shadow-5 d-flex justify-content-center">
                        <div className="d-flex justify-content-center">
                            <QRCode value={roomID===""?"":`${baseUrl}recv/`+roomID} className="card-img-top m-5" />
                        </div>
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
        </div>
    );
}

export default CreateRoom