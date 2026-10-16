package main

import (
	"context"
	"strings"
	"testing"
)

func TestRootCommandIncludesPrivacyEraseUserCommand(t *testing.T) {
	root := newRootCommand()
	command, _, err := root.Find([]string{"privacy:erase-user"})
	if err != nil || command == nil {
		t.Fatalf("find privacy:erase-user: command=%v err=%v", command, err)
	}
}

// 用法契约先于连接配置：缺 actor、目标二选一不成立时，命令必须在触碰数据库
// 之前就失败。
func TestPrivacyEraseUserUsageContract(t *testing.T) {
	opts := privacyEraseUserOptions{ActorUserID: 0, TargetID: 1}
	err := runPrivacyEraseUserCommand(context.Background(), opts, newPrivacyEraseUserCommand())
	if err == nil || !strings.Contains(err.Error(), "--actor-user-id") {
		t.Fatalf("missing actor error = %v", err)
	}

	opts = privacyEraseUserOptions{ActorUserID: 1}
	err = runPrivacyEraseUserCommand(context.Background(), opts, newPrivacyEraseUserCommand())
	if err == nil || !strings.Contains(err.Error(), "--user-id") {
		t.Fatalf("target ambiguity error = %v", err)
	}

	opts = privacyEraseUserOptions{ActorUserID: 1, TargetID: 2, TargetEmail: "a@b.c"}
	err = runPrivacyEraseUserCommand(context.Background(), opts, newPrivacyEraseUserCommand())
	if err == nil || !strings.Contains(err.Error(), "--user-id") {
		t.Fatalf("both-targets error = %v", err)
	}

	// 缺数据库连接串也要在 actor/target 契约之后才报，且报得可读。
	// CI 的 Go gate 会给整个测试进程 export DATABASE_URL（隔离测试库），
	// 这里显式清空，保证这条分支在任何环境下都触达「连接串为空」的错误。
	t.Setenv("DATABASE_URL", "")
	opts = privacyEraseUserOptions{ActorUserID: 1, TargetID: 2}
	err = runPrivacyEraseUserCommand(context.Background(), opts, newPrivacyEraseUserCommand())
	if err == nil || !strings.Contains(err.Error(), "database url") {
		t.Fatalf("missing database url error = %v", err)
	}
}
