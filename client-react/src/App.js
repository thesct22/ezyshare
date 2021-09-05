import React from 'react';
import {BrowserRouter,Route,Switch} from 'react-router-dom';


import CreateRoom from "./components/CreateRoom"
import Receive from './components/Receive';
import Send from "./components/Send";

function App() {
  return (
    <div className="App">
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
