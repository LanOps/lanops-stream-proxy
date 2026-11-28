package config

type Config struct {
	DbPath               string
	ObsWebSocketAddress  string
	ObsWebSocketPassword string
	ObsProxySceneName    string
	ObsProxySceneUuid    string
	ObsCloseWhenInactive bool
	MediaMtxApiAddress   string
	MediaMtxRtmpAddress  string
	ApiAdminUsername     string
	ApiAdminPassword     string
	ApiPort              string
}
