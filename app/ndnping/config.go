package ndnping

import (
	"time"
)

type TaskConfig struct {
	Face   FaceConfig
	Client *ClientConfig
	Server *ServerConfig
}

type FaceConfig struct {
	Remote string
	Local  string
}

type ClientConfig struct {
	Patterns []ClientPattern
	Interval time.Duration
}

type ClientPattern struct {
	Prefix string
	Repeat int
}

type ServerConfig struct {
	Patterns []ServerPattern
	Nack     bool
}

type ServerPattern struct {
	Prefix     string
	PayloadLen int
	Suffix     string
}
