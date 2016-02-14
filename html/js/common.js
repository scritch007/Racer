
//This should be kept synced with the types/message.go file
var EnumMessage = {
	CONTROL: 0,
	MOVE: 1
};

var EnumControl={
	NewInstance: 0,
	StartClientSession: 1,
	StartServerSession: 2,
	NewPlayerConnected: 3,
	CreateSession: 4,
	ConnectSession: 5,
};

// NewInstance
function NewInstanceConfig(websocket, multiplayer){
	this.w = websocket; // Set to false to switch to WebRtc
	this.m = multiplayer; // Multiple player can connect
}

var EnumMove = {
	Moving: 0,
	Collision: 1,
}