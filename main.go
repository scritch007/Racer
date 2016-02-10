package main

import (
	"flag"
	"html/template"
	"image/png"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/scritch007/Racer/types"
	"github.com/scritch007/go-tools"
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
		tools.LOG_ERROR.Print("upgrade:", err)
		return
	}

	//TODO generate random ID
	id := "123456"
	sessionMessage := types.Message{
		Type:    types.EnumMessageControl,
		SubType: types.EnumControlNewInstance,
		Message: id,
	}
	m, err := sessionMessage.ToString()
	if err != nil {
		tools.LOG_ERROR.Println("Failed to serialize message ", m)
	}

	c.WriteMessage(1, []byte(m))
	serverConnections[id] = c

	//Send id to the server
	_, _, err = c.ReadMessage()
	if err != nil {
		tools.LOG_ERROR.Println("read:", err)
		return
	}

	defer func() {
		c.Close()
		delete(serverConnections, id)
	}()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			tools.LOG_ERROR.Println("read:", err)
			break
		}
		tools.LOG_ERROR.Printf("recv: %s", message)
		err = c.WriteMessage(mt, []byte(`{"type":"message", "data":`+string(message)+"}"))
		if err != nil {
			tools.LOG_ERROR.Println("write:", err)
			break
		}
	}
}

func clientWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		tools.LOG_ERROR.Print("upgrade:", err)
		return
	}
	defer c.Close()
	serverSocket, _ := serverConnections[id]

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			tools.LOG_ERROR.Println("read:", err)
			break
		}
		newMssage, err := types.MessageFromString(string(message))
		if nil != err {
			tools.LOG_ERROR.Println("Failed to deserialize message " + string(message) + " " + err.Error())
			continue
		}

		//TODO add the ClientId in here
		m, err := newMssage.ToString()
		if nil != err {
			tools.LOG_ERROR.Println("Couldn't serialize message ", err)
			continue
		}

		serverSocket.WriteMessage(mt, []byte(m))

		tools.LOG_ERROR.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			tools.LOG_ERROR.Println("write:", err)
			break
		}
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("./html/index.html")
	if err != nil {
		panic(err)
	}
	values := struct {
		WSocketURL string
		QRCodeUrl  string
	}{
		WSocketURL: "ws://" + r.Host + "/server",
		QRCodeUrl:  "qrcode/123456.png",
	}
	err = tmpl.Execute(w, values)
	if nil != err {
		tools.LOG_ERROR.Fatalln(err.Error())
	}
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

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	qrcode, err := qr.Encode("http://"+r.Host+"/client/"+id, qr.L, qr.Auto)
	if err != nil {
		tools.LOG_ERROR.Println(err)
	} else {
		qrcode, err = barcode.Scale(qrcode, 100, 100)
		if err != nil {
			tools.LOG_ERROR.Println(err)
		} else {
			png.Encode(w, qrcode)
		}
	}

}

func serveFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	filePath := vars["file"]
	http.ServeFile(w, r, "./html/"+filePath)
}

func main() {
	tools.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	flag.Parse()
	tools.LOG_ERROR.SetFlags(0)
	r := mux.NewRouter()
	r.HandleFunc("/client/{id}", client)
	r.HandleFunc("/server", serverWS)
	r.HandleFunc("/client.ws/{id}", clientWS)
	r.HandleFunc("/qrcode/{id}.png", qrcodeHandler)
	r.HandleFunc("/static/{file:.*}", serveFile)
	r.HandleFunc("/", home)
	http.Handle("/", r)
	tools.LOG_ERROR.Fatal(http.ListenAndServe(*addr, nil))
}
