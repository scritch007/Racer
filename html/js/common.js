
//This should be kept synced with the types/message.go file
var EnumMessage = {
  CONTROL: 1,
  MOVE: 2
};

var EnumControl={
  NewInstance: 1,
  StartClientSession: 2,
  StartServerSession: 3,
  NewPlayerConnected: 4,
  CreateSession: 5,
  ConnectSession: 6,
};

// NewInstance
function NewInstanceConfig(websocket, multiplayer){
  this.w = websocket; // Set to false to switch to WebRtc
  this.m = multiplayer; // Multiple player can connect
}

var EnumMove = {
  Moving: 1,
  Collision: 2,
}


function Connection(wspath, websocket, multiplayer, server){
// Initialise DataChannel.js
var datachannel = {};
if (!websocket){
  datachannel = new DataChannel();
}

datachannel.userid = userid;

var ws;
ws = new WebSocket(wspath);
ws.onopen = function(evt) {
  console.log("OPEN");
  if (server){
    var newInstance = new NewInstanceConfig(websocket, multiplayer);
    ws.send(JSON.stringify({
      t: EnumMessage.CONTROL,
      s: EnumControl.NewInstance,
      m: JSON.stringify(newInstance),
    }));
  }else{
    self.onopen();
  }
}
ws.onclose = function(evt) {
  console.log("CLOSE");
  ws = null;
}
ws.onerror = function(evt) {
  console.log("ERROR: " + evt.data);
}
  var self = this;
  this.onmessage = function(message){
    //This should be overwritten by the caller
  }

ws.onmessage = function(evt) {
  //For now. This will be overwritten later
  console.log(evt.data);
  self.onmessage(evt.data);
}

datachannel.openSignalingChannel = function(config) {

  var channel = config.channel || this.channel || "default-channel";
  var xhrErrorCount = 0;

  var socket = {
    send: function(message) {
      ws.send(JSON.stringify(message));
    },
    channel: channel
  };
  if (!websocket){
    ws.onmessage = function(evt) {
      var jsonObject = JSON.parse(evt.data);
      console.log(evt.data);
      config.onmessage(jsonObject);
    }
  }
  if (config.onopen) {
    setTimeout(config.onopen, 1);
  }
  return socket;
}


var onCreateChannel = function() {
  var channelName = cleanChannelName(channelInput.value);

  if (!channelName) {
    console.log("No channel name given");
    return;
  }

  disableConnectInput();

  datachannel.open(channelName);
};

var onJoinChannel = function() {
  var channelName = cleanChannelName(channelInput.value);

  if (!channelName) {
    console.log("No channel name given");
    return;
  }

  disableConnectInput();

  // Search for existing data channels
  datachannel.connect(channelName);
};

  datachannel.onmessage = function(message){
    self.onmessage(message);
  }

  this.openchannel = function(channelName){
    datachannel.open(channelName);
    console.log("Openning channel " + channelName);
  }

  this.connectchannel = function(channelName){
    if (!websocket){
      datachannel.connect(channelName);
      console.log("Connecting channel " + channelName);
    }
  }
  this.onopen = function(){

  };

  this.send = function(message){
    //Depending on the type use the websocket or the webrtc channel
    if (websocket){
      if (ws)
        ws.send(message);
    }else{
      datachannel.send(message);
    }
  }

}