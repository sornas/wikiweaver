package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PORT                            = 4242
	CONSOLE_SOCKET_PATH             = "/tmp/ww-console.sock"
	CODE_LENGTH                     = 4
	LOBBY_IDLE_TIME_BEFORE_SHUTDOWN = 15 * time.Minute
	HISTORY_SEND_INTERVAL           = 200 * time.Millisecond
)

type GlobalState struct {
	Lobbies map[string]*Lobby
}

var globalState GlobalState

type WebClient struct {
	conn   *websocket.Conn
	isHost bool
}

type Lobby struct {
	Code                string
	WebClients          []*WebClient
	LastInteractionTime time.Time
	StartTime           time.Time
	StartPage           string
	GoalPage            string
	History             []PageToWebMessage
}

func (l *Lobby) close() {
	for _, wc := range l.WebClients {
		wc.conn.Close()
	}
}

func (l *Lobby) hasHost() bool {
	for _, wc := range l.WebClients {
		if wc.isHost {
			return true
		}
	}

	return false
}

func (l *Lobby) removeWebClient(wcToRemove *WebClient) error {
	index := -1
	for i, wc := range l.WebClients {
		if wc == wcToRemove {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("internal server error: web client %v not found in list %v", wcToRemove, l.WebClients)
	}

	length := len(l.WebClients)

	l.WebClients[index] = l.WebClients[length-1]
	l.WebClients[length-1] = nil
	l.WebClients = l.WebClients[:length-1]

	return nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func generateCode() string {
	CAPITAL_LETTERS := "ABCDEFGHIJKLMNOPQRSTUVXYZ"

	b := make([]byte, CODE_LENGTH)

	for i := range b {
		b[i] = CAPITAL_LETTERS[rand.Intn(len(CAPITAL_LETTERS))]
	}

	return string(b)
}

func generateUniqueCode() string {

	code := generateCode()

	for {
		if _, ok := globalState.Lobbies[code]; !ok {
			break
		}

		code = generateCode()
	}

	return code
}

func lobbyCleaner() {
	for {
		time.Sleep(LOBBY_IDLE_TIME_BEFORE_SHUTDOWN)

		for code, lobby := range globalState.Lobbies {
			if time.Now().After(lobby.LastInteractionTime.Add(LOBBY_IDLE_TIME_BEFORE_SHUTDOWN)) {
				idleTime := time.Since(lobby.LastInteractionTime).Round(time.Second)
				log.Printf("lobby %s idle for %s, closing", code, idleTime)
				lobby.close()
				delete(globalState.Lobbies, code)
			}
		}
	}
}

func handleWebLobbyCreate(w http.ResponseWriter, r *http.Request) {

	code := generateUniqueCode()

	globalState.Lobbies[code] = &Lobby{Code: code, LastInteractionTime: time.Now()}

	log.Printf("web client %s created lobby %s", r.RemoteAddr, code)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(code))
}

type Message struct {
	Type string
}

type PongMessage struct {
	Message
}

type StartMessage struct {
	Message
	StartPage string
	GoalPage  string
}

func handleWebLobbyJoin(w http.ResponseWriter, r *http.Request) {

	code := r.URL.Query().Get("code")

	lobby := globalState.Lobbies[code]

	if lobby == nil {
		log.Printf("%s tried to join non existent lobby: %s", r.RemoteAddr, code)
		return
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	shouldBeHost := !lobby.hasHost()
	wc := &WebClient{conn: conn, isHost: shouldBeHost}
	lobby.WebClients = append(lobby.WebClients, wc)

	if shouldBeHost {
		log.Printf("web client %s joined lobby %s as host", conn.RemoteAddr(), lobby.Code)
	} else {
		log.Printf("web client %s joined lobby %s as spectator", conn.RemoteAddr(), lobby.Code)
	}

	lobby.LastInteractionTime = time.Now()

	go webClientListener(lobby, wc)

	if !lobby.StartTime.IsZero() {
		// Lobby has already started, we have history to send
		go sendHistory(lobby, wc)
	}
}

func sendHistory(lobby *Lobby, wc *WebClient) {
	log.Printf("sending history from lobby %s to web client %s", lobby.Code, wc.conn.RemoteAddr())

	startMsg := StartMessage{
		Message: Message{
			Type: "start",
		},
		StartPage: lobby.StartPage,
		GoalPage:  lobby.GoalPage,
	}

	err := wc.conn.WriteJSON(startMsg)
	if err != nil {
		log.Printf("failed to send history: %s", err)
		return
	}

	for _, msg := range lobby.History {
		time.Sleep(HISTORY_SEND_INTERVAL)
		err := wc.conn.WriteJSON(msg)
		if err != nil {
			log.Printf("failed to send history: %s", err)
			break
		}
	}
}

func webClientListener(lobby *Lobby, wc *WebClient) {
	defer wc.conn.Close()

	for {
		_, buf, err := wc.conn.ReadMessage()
		if err != nil {
			log.Printf("web client %s disconnected from lobby %s\n", wc.conn.RemoteAddr(), lobby.Code)

			err = lobby.removeWebClient(wc)
			if err != nil {
				log.Printf("failed to remove web client: %s", err)
			}

			return
		}

		var msg Message
		err = json.Unmarshal(buf, &msg)
		if err != nil {
			log.Printf("failed to unmarshal message: %s", err)
			continue
		}

		switch msg.Type {
		case "ping":
			pongMessage := PongMessage{
				Message: Message{
					Type: "pong",
				},
			}

			err = wc.conn.WriteJSON(pongMessage)
			if err != nil {
				log.Printf("failed to respond with pong: %s", err)
				continue
			}
		}
	}
}

type LobbyStatusResponse struct {
	Active bool
}

func handleWebLobbyStatus(w http.ResponseWriter, r *http.Request) {

	code := r.URL.Query().Get("code")

	lobby := globalState.Lobbies[code]

	response := LobbyStatusResponse{Active: lobby != nil}

	msg, err := json.Marshal(response)
	if err != nil {
		log.Printf("failed to marshal status response: %s", err)
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write(msg)
}

type StartMessageFromWeb struct {
	Code      string
	StartPage string
	GoalPage  string
}

func handleWebLobbyStart(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading extension request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		var startMessageFromWeb StartMessageFromWeb
		err = json.Unmarshal(body, &startMessageFromWeb)
		if err != nil {
			log.Printf("failed to parse start message from web: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		code := startMessageFromWeb.Code

		lobby := globalState.Lobbies[code]

		if lobby == nil {
			log.Printf("failed to start lobby %s: lobby does not exist", code)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte{})
			return
		}

		if !lobby.StartTime.IsZero() {
			log.Printf("failed to start lobby %s: lobby already started", code)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte{})
			return
		}

		lobby.StartTime = time.Now()
		lobby.StartPage = startMessageFromWeb.StartPage
		lobby.GoalPage = startMessageFromWeb.GoalPage
		lobby.LastInteractionTime = time.Now()

		log.Printf("web client %s started lobby %s: '%s' -> '%s'", r.RemoteAddr, code, lobby.StartPage, lobby.GoalPage)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
	}
}

type PageFromExtMessage struct {
	Code     string
	Username string
	Page     string
	Backmove bool
}

type PageToWebMessage struct {
	Message
	Username  string
	Page      string
	TimeAdded int64
	Backmove  bool
}

func handleExtPage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("error reading extension request: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		var pageFromExtMessage PageFromExtMessage
		err = json.Unmarshal(body, &pageFromExtMessage)
		if err != nil {
			log.Printf("failed to parse message from extension: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		code := pageFromExtMessage.Code

		lobby := globalState.Lobbies[code]
		if lobby == nil {
			log.Printf("refusing to forward page to lobby %s: lobby not found", code)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		if lobby.StartTime.IsZero() {
			log.Printf("refusing to forward page to lobby %s: lobby is not started", code)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		lobby.LastInteractionTime = time.Now()

		pageToWebMessage := PageToWebMessage{
			Message: Message{
				Type: "page",
			},
			Username:  pageFromExtMessage.Username,
			Page:      pageFromExtMessage.Page,
			TimeAdded: time.Since(lobby.StartTime).Milliseconds(),
			Backmove:  pageFromExtMessage.Backmove,
		}

		lobby.History = append(lobby.History, pageToWebMessage)

		log.Printf("forwarding page from extension %s to %d web clients in lobby %s: %v", r.RemoteAddr, len(lobby.WebClients), code, pageToWebMessage)

		for _, wc := range lobby.WebClients {
			err = wc.conn.WriteJSON(pageToWebMessage)
			if err != nil {
				log.Printf("failed to forward message to web client %s: %s", wc.conn.RemoteAddr(), err)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
	}
}

func main() {

	dev := false
	for _, arg := range os.Args[1:] {
		if arg == "--dev" {
			dev = true
		}
	}

	globalState = GlobalState{Lobbies: make(map[string]*Lobby)}

	go ConsoleListener()
	go lobbyCleaner()

	http.HandleFunc("/api/web/lobby/create", handleWebLobbyCreate)
	http.HandleFunc("/api/ws/web/lobby/join", handleWebLobbyJoin)
	http.HandleFunc("/api/web/lobby/status", handleWebLobbyStatus)
	http.HandleFunc("/api/web/lobby/start", handleWebLobbyStart)

	http.HandleFunc("/api/ext/page", handleExtPage)

	address := "0.0.0.0"
	if dev {
		address = "localhost"
	}

	address = fmt.Sprintf("%s:%d", address, PORT)

	log.Printf("listening on %s", address)

	var err error

	if dev {
		err = http.ListenAndServe(address, nil)
	} else {
		err = http.ListenAndServeTLS(address, "/secrets/ssl_certificate.txt", "/secrets/ssl_privatekey.txt", nil)
	}

	if err != nil {
		log.Fatalf("listen error: %s", err)
	}
}

func ConsoleListener() {

	_, err := os.Stat(CONSOLE_SOCKET_PATH)
	if !os.IsNotExist(err) {
		os.Remove(CONSOLE_SOCKET_PATH)
	}

	listener, err := net.Listen("unix", CONSOLE_SOCKET_PATH)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	defer listener.Close()

	log.Printf("developer console listening on %s", CONSOLE_SOCKET_PATH)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}

		log.Printf("received console connection")
		go ConsoleHandler(conn)
	}
}

func ConsoleHandler(conn net.Conn) {
	defer conn.Close()

	for {
		buf := make([]byte, 512)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("console connection closed")
			return
		}

		cmd := strings.Fields(string(buf[:n]))

		if len(cmd) > 0 {
			if cmd[0] != "newpage" {
				conn.Write([]byte("unknown command\n"))
				continue
			}

			if len(cmd) != 4 {
				conn.Write([]byte("usage: newpage <lobby> <username> <page>\n"))
				continue
			}

			lobby := globalState.Lobbies[cmd[1]]

			if lobby == nil {
				conn.Write([]byte("lobby does not exist\n"))
				continue
			}

			lobby.LastInteractionTime = time.Now()

			// msg := PageToWebMessage{Username: cmd[2], Page: cmd[3]}
			// lobby.Host.WriteJSON(msg)
		}
	}
}
