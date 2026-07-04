package migrations

import (
	"embed"
	"io/fs"
)

//go:embed *.sql
var files embed.FS

// Files 返回编译进二进制的 Goose SQL 迁移文件系统。
func Files() fs.FS {
	return files
}
