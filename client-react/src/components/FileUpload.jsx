import React, { useState } from "react";

const UploadFiles = (props) => {
    var [selectedFile, setSelectedFile] = useState(undefined);
    const selectFile = (event) => {
      if(event.target.files[0]!==undefined&&event.target.files[0].size<10485770)
        setSelectedFile(event.target.files[0]);
      else
        setSelectedFile(undefined)
    };

    const upload = () => {
      if(selectedFile!==undefined)
        props.addfile(selectedFile)
    }

    return (
     <div>

      <h4> Maximum size of 10MB</h4>

      <label className="btn btn-default">
        <input type="file" onChange={selectFile}/>
      </label>

      <button
        className="btn btn-success"
        disabled={!selectedFile || props.disabled}
        onClick={upload}
      >
        Upload
      </button>

    </div>
  );
};

export default UploadFiles;