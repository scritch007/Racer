package main

import (
	"flag"
	"html/template"
	"image/png"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"

	"github.com/jmcvetta/randutil"

	"github.com/scritch007/Racer/types"
	"github.com/scritch007/go-tools"
)

var addr = flag.String("addr", ":8080", "http service address")
var debug = flag.Bool("debug", false, "Turn into debug mode")

var upgrader = websocket.Upgrader{} // use default options

type serverConnection struct {
	server  *websocket.Conn
	clients map[string]*websocket.Conn
	config  *types.NewInstanceConfig
}

var serverConnections map[string]*serverConnection

func init() {
	serverConnections = make(map[string]*serverConnection)
}

func serverWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		tools.LOG_ERROR.Print("upgrade:", err)
		return
	}

	newServerConnection := new(serverConnection)
	newServerConnection.server = c
	newServerConnection.clients = make(map[string]*websocket.Conn)
	newServerConnection.config = nil

	serverConnections[id] = newServerConnection

	//Send id to the server
	_, _, err = c.ReadMessage()
	if err != nil {
		tools.LOG_ERROR.Println("read:", err)
		return
	}

	defer func() {
		c.Close()
		//TODO notify all the clients + cleanup everyting correctly
		delete(serverConnections, id)
	}()

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			tools.LOG_ERROR.Println("read:", err)
			break
		}
		tools.LOG_DEBUG.Printf("Server recv: %s", message)
		mfs, err := types.MessageFromString(string(message))
		if nil != err {
			tools.LOG_ERROR.Println("Error deserializing message ", err)
			continue
		}

		if mfs.Type == types.EnumMessageControl {
			if mfs.SubType == types.EnumControlNewInstance {
				nic, err := types.NewInstanceConfigFromString(mfs.Message)
				if nil != err {
					tools.LOG_ERROR.Println("Error deserializing new Instance ", err)
					continue
				}
				//Check for configuration
				newServerConnection.config = nic
			}
		} else if mfs.Type == types.EnumMessageMove {
			if 0 == len(newServerConnection.clients) {
				tools.LOG_DEBUG.Println("Client is still empty")
				continue
			}
			for _, client := range newServerConnection.clients {
				client.WriteMessage(mt, message)
			}
		} else {
			tools.LOG_ERROR.Println("Invalid type")
		}

	}
}

func clientWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	subid := vars["subid"]

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		tools.LOG_ERROR.Print("upgrade:", err)
		return
	}
	serverSocket, found := serverConnections[id]
	if !found {
		tools.LOG_ERROR.Println("Couldn't find server side")
		c.Close()
		return
	}
	defer func() {
		c.Close()
		//TODO notify all the clients + cleanup everyting correctly
		delete(serverConnections.clients, subid)
	}()

	if nil != serverSocket.config {
		tools.LOG_ERROR.Println("Opening connection on an empty configuration. Closing connection")
		return
	}

	if 0 != len(serverSocket.clients) && !serverSocket.config.Multiplayer {
		_, _, _ = c.ReadMessage()
		c.WriteMessage(0, []byte("Someone is already there"))
		return
	}
	serverSocket.clients[subid] = c

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

		serverSocket.server.WriteMessage(mt, []byte(m))
		//tools.LOG_DEBUG.Printf("recv: %s", message)
	}
}

func getProto(r *http.Request, websocket bool) string {
	var resProto string
	if websocket {
		resProto = "ws"
	} else {
		resProto = "http"
	}
	proto := r.Header.Get("X-Forwarded-Proto")
	tools.LOG_DEBUG.Println(proto + r.Proto)
	if 0 != len(proto) {
		if proto == "https" {
			return resProto + "s"
		}
	} else if strings.Contains(r.Proto, "HTTPS") {
		resProto = resProto + "s"
	}
	return resProto
}

func server(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	tmpl, err := template.ParseFiles("./html/index.html")
	if err != nil {
		panic(err)
	}
	socketProto := getProto(r, true)
	values := struct {
		WSocketURL string
		QRCodeUrl  string
	}{
		WSocketURL: socketProto + "://" + r.Host + "/server.ws/" + id,
		QRCodeUrl:  "/qrcode/" + id + ".png",
	}
	err = tmpl.Execute(w, values)
	if nil != err {
		tools.LOG_ERROR.Fatalln(err.Error())
	}
}

func clientRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var subid string
	if *debug {
		subid = "7890"
	} else {
		subid, _ = randutil.AlphaString(20)
	}

	proto := getProto(r, false)

	url := proto + "://" + r.Host + "/client/" + id + "/" + subid
	tools.LOG_DEBUG.Println("Redirecting to " + url)
	http.Redirect(w, r, url, http.StatusFound)
}

func client(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	subid := vars["subid"]

	tmpl, err := template.ParseFiles("./html/client.html")
	if err != nil {
		panic(err)
	}
	socketProto := getProto(r, true)
	tmpl.Execute(w, socketProto+"://"+r.Host+"/client.ws/"+id+"/"+subid)
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var proto = "http"
	if strings.Contains(r.Proto, "HTTPS") {
		proto = "httpss"
	}

	qrcode, err := qr.Encode(proto+"://"+r.Host+"/client/"+id, qr.L, qr.Auto)
	if err != nil {
		tools.LOG_ERROR.Println(err)
	} else {
		qrcode, err = barcode.Scale(qrcode, 150, 150)
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

func home(w http.ResponseWriter, r *http.Request) {
	var id string
	if *debug {
		id = "123456"
	} else {
		id, _ = randutil.AlphaString(20)
	}
	//Redirect the server to a new instance
	http.Redirect(w, r, "/server/"+id, http.StatusFound)
}

func main() {
	tools.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	flag.Parse()
	tools.LOG_ERROR.SetFlags(0)
	r := mux.NewRouter()
	r.HandleFunc("/client/{id}", clientRedirect)
	r.HandleFunc("/client/{id}/{subid}", client)
	r.HandleFunc("/server/{id}", server)
	r.HandleFunc("/server.ws/{id}", serverWS)
	r.HandleFunc("/client.ws/{id}/{subid}", clientWS)
	r.HandleFunc("/qrcode/{id}.png", qrcodeHandler)
	r.HandleFunc("/static/{file:.*}", serveFile)
	r.HandleFunc("/", home)
	http.Handle("/", r)
	tools.LOG_ERROR.Fatal(http.ListenAndServe(*addr, nil))
}
