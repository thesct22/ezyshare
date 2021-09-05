import React from 'react';
import {BrowserRouter,Route,Switch} from 'react-router-dom';

import "./styles.css";

import CreateRoom from "./components/CreateRoom"
import Receive from './components/Receive';
import Send from "./components/Send";
import DarkModeToggler from './components/DarkModeToggler';

function App() {
  return (
    <div className="App">
      <div className="row bg-primary p-2">
        <div className="col-md-10">
          <h1 className="pl-4">EzyShare</h1>
        </div>
        <div className="col-md-2 d-flex align-items-center justify-content-center">
          <DarkModeToggler/>
        </div>
      </div>
      <BrowserRouter basename={process.env.PUBLIC_URL}>
        <Switch>
          <Route exact path="/" component={CreateRoom}></Route>
          <Route path="/send/:roomID" component={Send}></Route>
          <Route path="/recv/:roomID" component={Receive}></Route>
        </Switch>
      </BrowserRouter>
    </div>
  );
}

export default App;
