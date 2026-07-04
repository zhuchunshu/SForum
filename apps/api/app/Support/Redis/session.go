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

func StorageConfig(addr string, password string) (redisstorage.Config, error) {
	host, port, err := ParseAddr(addr)
	if err != nil {
		return redisstorage.Config{}, err
	}

	return redisstorage.Config{
		Host:     host,
		Port:     port,
		Password: password,
	}, nil
}

func NewStorage(addr string, password string) (*redisstorage.Storage, error) {
	cfg, err := StorageConfig(addr, password)
	if err != nil {
		return nil, err
	}

	return redisstorage.New(cfg), nil
}
