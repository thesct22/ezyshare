import React, { useState } from "react";
import Dropzone from 'react-dropzone';


const UploadFiles = (props) => {
    var [selectedFile, setSelectedFile] = useState(undefined);
    const selectFile = (event) => {
      //console.log(event)
      if(event[0]!==undefined&&event[0].size<10485770)
        setSelectedFile(event[0]);
      else
        setSelectedFile(undefined)
    };

    const upload = () => {
      if(selectedFile!==undefined)
        props.addfile(selectedFile)
    }

    return (
     <div className="container col-sm-8 border m-sm-4">

      <h4>Upload File</h4>

      <div className="col-sm-12">
        <Dropzone onDrop={selectFile}>
          {({getRootProps, getInputProps}) => (
            <section className="w-100 d-flex align-items-center justify-content-center">
              <div {...getRootProps()} className="d-flex align-items-center justify-content-center w-100" style={{height: "200px", backgroundColor: "#BDBDBD"}}>
                <input {...getInputProps()} />
                <div className="row d-flex align-items-center justify-content-center text-center">
                  <p className="m-5">{"Drag 'n' drop some files here, or click to select files"}</p>
                </div>
              </div>
            </section>
          )}
        </Dropzone>
      </div>
      <div className="row">
        <div className="col">
          <p className="p-3">{"Maximum size should be 10MB"}</p>
        </div>
        <div className="col">
          <button className="btn btn-success" disabled={!selectedFile || props.disabled} onClick={upload}>
            Upload
          </button>
        </div>
      </div>
      
      


    </div>
  );
};

export default UploadFiles;