package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"image/png"
	"io/ioutil"
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

//GameConfig the game configuration
type GameConfig struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Game  string
}

var games []GameConfig

type serverConnection struct {
	server  *websocket.Conn
	clients map[string]*websocket.Conn
	config  *types.NewInstanceConfig
}

var serverConnections map[string]*serverConnection

func init() {
	serverConnections = make(map[string]*serverConnection)
	games = make([]GameConfig, 0, 10)
	files, err := ioutil.ReadDir("./games/")
	if nil != err {
		tools.LOG_ERROR.Printf("Couldn't find the folder")
	}
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if f.Name() == "html" || f.Name() == ".git" {
			//Then skip it
		} else {
			var c GameConfig
			configFileName := "./games/" + f.Name() + "/app.json"
			data, err := ioutil.ReadFile(configFileName)
			err = json.Unmarshal(data, &c)
			if nil != err {
				tools.LOG_ERROR.Fatalf("Couldn't read configuration %s, %s\n", configFileName, err.Error())
			} else {
				c.Game = f.Name()
				games = games[:len(games)+1]
				games[len(games)-1] = c
			}
		}
	}
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
	tools.LOG_DEBUG.Printf("Adding new server connection %s\n", id)

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
			tools.LOG_ERROR.Println("Invalid json received error "+string(message), err)
			continue
		}
		if 0 == mfs.Type {
			// Forward message... This is a valid json but not correct message type
			for _, client := range newServerConnection.clients {
				// Writing to client
				tools.LOG_DEBUG.Println("Writing to client")
				client.WriteMessage(mt, message)
			}
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
		} else if mfs.Type == types.EnumMessageCustom {
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
		tools.LOG_ERROR.Printf("Couldn't find server side %s\n", id)
		c.Close()
		return
	}
	defer func() {
		c.Close()
		//TODO notify all the clients + cleanup everyting correctly
		delete(serverSocket.clients, subid)
	}()

	if nil == serverSocket.config {
		tools.LOG_ERROR.Println("Opening connection on an empty configuration. Closing connection")
		return
	}

	if 0 != len(serverSocket.clients) && !serverSocket.config.Multiplayer {
		_, _, _ = c.ReadMessage()
		c.WriteMessage(0, []byte("Someone is already there"))
		return
	}
	serverSocket.clients[subid] = c
	//Notify Server that the client arrived

	newClientMessage := types.Message{
		Type:    types.EnumMessageControl,
		SubType: types.EnumControlStartClientSession,
		Message: subid,
	}
	ncmj, err := newClientMessage.ToString()
	if nil != err {
		tools.LOG_ERROR.Println("Failed to serialize message ", err)
		return
	}
	serverSocket.server.WriteMessage(1, []byte(ncmj))

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
		if 0 == newMssage.Type {
			// Invalid message type forward to server
			serverSocket.server.WriteMessage(mt, message)
			continue
		}
		if !serverSocket.config.Websocket && newMssage.Type == types.EnumMessageControl {
			tools.LOG_ERROR.Println("Received move message from client but this should be the case...")
			continue
		}

		newMssage.ClientId = subid
		m, err := newMssage.ToString()
		if nil != err {
			tools.LOG_ERROR.Println("Couldn't serialize message ", err)
			continue
		}
		tools.LOG_DEBUG.Println("Client " + subid + " send :" + m)
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
	game := vars["game"]

	tmpl, err := template.ParseFiles("./games/" + game + "/index.html")
	if err != nil {
		panic(err)
	}
	socketProto := getProto(r, true)
	values := struct {
		WSocketURL string
		QRCodeURL  string
		UserID     string
		Game       string
		ClientURL  string
	}{
		WSocketURL: socketProto + "://" + r.Host + "/" + game + "/server.ws/" + id,
		QRCodeURL:  "/" + game + "/qrcode/" + id + ".png",
		UserID:     id,
		Game:       game,
		ClientURL:  getProto(r, false) + "://" + r.Host + "/" + game + "/client/" + id,
	}
	err = tmpl.Execute(w, values)
	if nil != err {
		tools.LOG_ERROR.Fatalln(err.Error())
	}
}

func clientRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	game := vars["game"]
	var subid string
	if *debug {
		subid = "7890"
	} else {
		subid, _ = randutil.AlphaString(20)
		//We should ensure that we didn't already random this string for another client for this session...
	}

	proto := getProto(r, false)

	url := proto + "://" + r.Host + "/" + game + "/client/" + id + "/" + subid
	tools.LOG_DEBUG.Println("Redirecting to " + url)
	http.Redirect(w, r, url, http.StatusFound)
}

func client(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	subid := vars["subid"]
	game := vars["game"]

	tmpl, err := template.ParseFiles("./games/" + game + "/client.html")
	if err != nil {
		panic(err)
	}
	socketProto := getProto(r, true)
	values := struct {
		WebSocketURL string
		UserID       string
		Game         string
	}{
		WebSocketURL: socketProto + "://" + r.Host + "/" + game + "/client.ws/" + id + "/" + subid,
		UserID:       subid,
		Game:         game,
	}
	err = tmpl.Execute(w, values)
	if nil != err {
		tools.LOG_ERROR.Println("Failed to render template ", err)
	}
}

func qrcodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	game := vars["game"]

	var proto = "http"
	if strings.Contains(r.Proto, "HTTPS") {
		proto = "httpss"
	}

	qrcode, err := qr.Encode(proto+"://"+r.Host+"/"+game+"/client/"+id, qr.L, qr.Auto)
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
	game := vars["game"]
	if game == "html" {
		http.ServeFile(w, r, "./"+game+"/"+filePath)
	} else {
		http.ServeFile(w, r, "./games/"+game+"/"+filePath)
	}

}

func gameHome(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	game := vars["game"]
	var id string
	if *debug {
		id = "123456"
	} else {
		id, _ = randutil.AlphaString(20)
	}
	//Redirect the server to a new instance
	http.Redirect(w, r, "/"+game+"/server/"+id, http.StatusFound)
}

func home(w http.ResponseWriter, r *http.Request) {
	values := struct {
		Items []GameConfig
	}{
		Items: games,
	}
	tmpl, err := template.ParseFiles("html/index.html")
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, values)
	if nil != err {
		tools.LOG_ERROR.Println("Failed to render template ", err)
	}
}

func main() {
	tools.LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)

	flag.Parse()
	tools.LOG_ERROR.SetFlags(0)
	r := mux.NewRouter()
	r.HandleFunc("/{game}/client/{id}", clientRedirect)
	r.HandleFunc("/{game}/client/{id}/{subid}", client)
	r.HandleFunc("/{game}/server/{id}", server)
	r.HandleFunc("/{game}/server.ws/{id}", serverWS)
	r.HandleFunc("/{game}/client.ws/{id}/{subid}", clientWS)
	r.HandleFunc("/{game}/qrcode/{id}.png", qrcodeHandler)
	r.HandleFunc("/{game}/static/{file:.*}", serveFile)
	r.HandleFunc("/{game}/", gameHome)
	r.HandleFunc("/", home)
	http.Handle("/", r)
	tools.LOG_ERROR.Fatal(http.ListenAndServe(*addr, nil))
}
