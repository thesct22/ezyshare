import React from 'react';

const CreateRoom =(props)=>{

    const create=async(e)=>{
        e.preventDefault()
        const resp=await fetch ("http://192.168.18.19:8000/make")
        const {room_id}=await resp.json();
        console.log(room_id)

        props.history.push(`/send/${room_id}`)
    }

    return(
        <div>
            <button onClick={create}>Create Room</button>
        </div>
    );
}

export default CreateRoom