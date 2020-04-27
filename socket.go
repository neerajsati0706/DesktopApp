package websocket

import (
	gv "Stefano/gv"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func reader(ConnWS *websocket.Conn) {
	gv.ConnWS = append(gv.ConnWS, ConnWS)
	for {
		// Read message from browser
		msgType, msg, err := ConnWS.ReadMessage()
		gv.MessageType = msgType
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(msgType)
		if err = ConnWS.WriteMessage(msgType, msg); err != nil {
			return
		}
	}
}
func Socket() {

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
		ws, err := upgrader.Upgrade(w, r, nil) // error ignored for sake of simplicity

		if err != nil {
			fmt.Println("Error")
		}
		fmt.Println("Successfuly...")
		reader(ws)
	})
	http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Ping Pong!!!!"))

		// http.ServeFile(w, r, "./websocket.html")
	})
	http.HandleFunc("/ws/test", testFunction)

}
func Notify(message string) {
	msg := []byte(message)

	for _, conn := range gv.ConnWS {
		err := conn.WriteMessage(1, msg)
		if err != nil {
			fmt.Println("Error")
		}
	}
}
func testFunction(w http.ResponseWriter, r *http.Request) {
	msg := []byte("Message from server")

	for _, conn := range gv.ConnWS {
		err := conn.WriteMessage(1, msg)
		if err != nil {
			fmt.Println("Error")
		}
	}

	w.Write([]byte("Ping Pong!!!!"))
}
