package redis

import (
	"fmt"
	"net"
	"strconv"

	redisstorage "github.com/gofiber/storage/redis/v3"
)

func ParseAddr(addr string) (string, int, error) {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("parse redis addr %q: %w", addr, err)
	}

	port, err := strconv.Atoi(portText)
	if err != nil {
		return "", 0, fmt.Errorf("parse redis port %q: %w", portText, err)
	}
	if host == "" || port <= 0 {
		return "", 0, fmt.Errorf("invalid redis addr %q", addr)
	}

	return host, port, nil
}

func NewStorage(addr string) (*redisstorage.Storage, error) {
	host, port, err := ParseAddr(addr)
	if err != nil {
		return nil, err
	}

	return redisstorage.New(redisstorage.Config{
		Host: host,
		Port: port,
	}), nil
}
