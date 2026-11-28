package obs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"

	"github.com/andreykaipov/goobs"
	"github.com/gorilla/websocket"
)

// fakeOBS is the smallest obs-websocket v5 server the obs package needs: it
// says hello, accepts the identify, and answers GetVersion and GetSceneList.
// Every connection reports a different scene UUID, which is what a freshly
// loaded scene collection looks like.
type fakeOBS struct {
	mu    sync.Mutex
	conns int
	live  []*websocket.Conn

	// inputKinds is what GetInputKindList reports. Defaults to a build with
	// the FreeType2 text source.
	inputKinds []string
}

// drop kills the server side of every open connection, the way OBS closing
// does. httptest.Server.Close does not touch hijacked WebSocket connections.
func (f *fakeOBS) drop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.live {
		_ = c.Close()
	}
	f.live = nil
}

func (f *fakeOBS) serve(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	f.mu.Lock()
	f.conns++
	f.live = append(f.live, conn)
	sceneUid := "scene-" + string(rune('a'+f.conns-1))
	f.mu.Unlock()

	if err := conn.WriteJSON(map[string]interface{}{
		"op": 0,
		"d":  map[string]interface{}{"obsWebSocketVersion": "5.5.2", "rpcVersion": 1},
	}); err != nil {
		return
	}

	for {
		var msg struct {
			Op int             `json:"op"`
			D  json.RawMessage `json:"d"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.Op == 1 { // Identify
			if err := conn.WriteJSON(map[string]interface{}{
				"op": 2,
				"d":  map[string]interface{}{"negotiatedRpcVersion": 1},
			}); err != nil {
				return
			}
			continue
		}
		if msg.Op != 6 { // Request
			continue
		}
		var req struct {
			RequestType string `json:"requestType"`
			RequestID   string `json:"requestId"`
		}
		_ = json.Unmarshal(msg.D, &req)

		resp := map[string]interface{}{
			"op": 7,
			"d": map[string]interface{}{
				"requestType":   req.RequestType,
				"requestId":     req.RequestID,
				"requestStatus": map[string]interface{}{"result": true, "code": 100},
			},
		}
		data := resp["d"].(map[string]interface{})

		switch req.RequestType {
		case "GetVersion":
			data["responseData"] = map[string]interface{}{"obsVersion": "30.2.3", "rpcVersion": 1}
		case "GetInputKindList":
			kinds := f.inputKinds
			if kinds == nil {
				kinds = []string{"ffmpeg_source", "text_ft2_source_v2"}
			}
			data["responseData"] = map[string]interface{}{"inputKinds": kinds}
		case "GetSceneList":
			data["responseData"] = map[string]interface{}{
				"currentProgramSceneName": "Proxy Scenes",
				"currentProgramSceneUuid": sceneUid,
				"scenes": []map[string]interface{}{
					{"sceneName": "Proxy Scenes", "sceneIndex": 0, "sceneUuid": sceneUid},
				},
			}
		default:
			data["requestStatus"] = map[string]interface{}{
				"result": false, "code": 600, "comment": "not implemented",
			}
		}
		if err := conn.WriteJSON(resp); err != nil {
			return
		}
	}
}

func (f *fakeOBS) connections() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conns
}

func newTestClient(t *testing.T, addr string) *Client {
	t.Helper()
	client, err := New(
		config.Config{ObsWebSocketAddress: addr, ObsProxySceneName: "Proxy Scenes"},
		nil,
		nil,
		make(chan channels.MsgCh, 64),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func noop(*goobs.Client) error { return nil }

// ping does one real round trip, like every sync and rotation call does.
func ping(obs *goobs.Client) error {
	_, err := obs.General.GetVersion()
	return err
}

// A dead OBS must not take the process with it, and the next call must dial
// again instead of erroring forever.
func TestWithObs_ReconnectsAfterServerRestart(t *testing.T) {
	f := &fakeOBS{}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	client := newTestClient(t, strings.TrimPrefix(srv.URL, "http://"))

	if err := client.withObs(ping); err != nil {
		t.Fatalf("first withObs: %v", err)
	}
	if client.cfg.ObsProxySceneUuid != "scene-a" {
		t.Fatalf("expected scene-a, got %q", client.cfg.ObsProxySceneUuid)
	}

	// OBS goes away mid-event.
	f.drop()
	time.Sleep(50 * time.Millisecond)
	if err := client.withObs(ping); err == nil {
		t.Fatal("expected the dead connection to fail a request")
	}
	if client.conn != nil {
		t.Fatal("expected the dead connection to be dropped")
	}

	// OBS comes back on the same address.
	srv2 := httptest.NewServer(http.HandlerFunc(f.serve))
	defer srv2.Close()
	client.cfg.ObsWebSocketAddress = strings.TrimPrefix(srv2.URL, "http://")

	if err := client.withObs(ping); err != nil {
		t.Fatalf("reconnect withObs: %v", err)
	}
	if got := f.connections(); got != 2 {
		t.Errorf("expected 2 connections, got %d", got)
	}
	// Scene UUIDs belong to the scene collection, so a reconnect has to
	// re-resolve the proxy scene rather than reuse the stale one.
	if client.cfg.ObsProxySceneUuid != "scene-b" {
		t.Errorf("expected scene-b after reconnect, got %q", client.cfg.ObsProxySceneUuid)
	}
}

// Dialling must be lazy: no OBS at startup is a retry, not a crash.
func TestWithObs_NoServerReturnsError(t *testing.T) {
	client := newTestClient(t, "127.0.0.1:1")
	if err := client.withObs(noop); err == nil {
		t.Fatal("expected an error when nothing is listening")
	}
	if client.conn != nil {
		t.Fatal("expected no connection to be kept")
	}
}
