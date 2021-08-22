import React, {useState} from 'react';
import {BrowserRouter,Route,Switch} from 'react-router-dom';

import "bootstrap/dist/css/bootstrap.min.css";


import CreateRoom from "./components/CreateRoom"
import Receive from './components/Receive';
import Room from "./components/Room";
import Send from "./components/Send";

function App() {
  return (
    <div className="App">
      <BrowserRouter>
        <Switch>
          <Route path="/" exact component={CreateRoom}></Route>
          <Route path="/room/:roomID" component={Room}></Route>
          <Route path="/send/:roomID" component={Send}></Route>
          <Route path="/recv/:roomID" component={Receive}></Route>
        </Switch>
      </BrowserRouter>
    </div>
  );
}

export default App;
