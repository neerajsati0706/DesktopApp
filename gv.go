package gv

import (
	"github.com/gorilla/websocket"
)

var VideoFolders []ReadWrite

var DeviceList = make([]DeviceListStruct, 0)

var CopyingData bool = false
var TotalUSB int = 0

var ConnWS []*websocket.Conn
var MessageType int

type DeviceListStruct struct {
	DeviceNo int
	Files    []ReadWrite
}

type ReadWrite struct {
	ReadFolder  string
	WriteFolder string
}
