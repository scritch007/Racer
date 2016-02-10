
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
};

var EnumMove = {
	Moving: 0,
	Collision: 1,
}