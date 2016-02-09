package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", ":8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

var serverConnections map[string]*websocket.Conn

func init() {
	serverConnections = make(map[string]*websocket.Conn)
}

func serverWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	id := "123456"

	c.WriteMessage(1, []byte(id))
	serverConnections[id] = c

	//Send id to the server
	_, _, err = c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	defer func() {
		c.Close()
		delete(serverConnections, id)
	}()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, []byte(`{"type":"message", "data":`+string(message)+"}"))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func clientWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	serverSocket, _ := serverConnections[id]

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		serverSocket.WriteMessage(mt, []byte(`{"type":"move", "data":`+string(message)+"}"))

		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("./html/index.html")
	if err != nil {
		panic(err)
	}
	tmpl.Execute(w, "ws://"+r.Host+"/server")
}

func client(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tmpl, err := template.ParseFiles("./html/client.html")
	if err != nil {
		panic(err)
	}
	tmpl.Execute(w, "ws://"+r.Host+"/client.ws/"+id)
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	r := mux.NewRouter()
	r.HandleFunc("/client/{id}", client)
	r.HandleFunc("/server", serverWS)
	r.HandleFunc("/client.ws/{id}", clientWS)
	r.HandleFunc("/", home)
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
