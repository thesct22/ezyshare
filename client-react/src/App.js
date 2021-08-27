import React from 'react';
import {BrowserRouter,Route,Switch} from 'react-router-dom';


import CreateRoom from "./components/CreateRoom"
import Receive from './components/Receive';
import Send from "./components/Send";
import "./assets/scss/custom.scss";

function App() {
  return (
    <div className="App">
      <BrowserRouter>
        <Switch>
          <Route path="/" exact component={CreateRoom}></Route>
          <Route path="/send/:roomID" component={Send}></Route>
          <Route path="/recv/:roomID" component={Receive}></Route>
        </Switch>
      </BrowserRouter>
    </div>
  );
}

export default App;
