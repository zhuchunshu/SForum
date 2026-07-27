package main

import (
	"strings"
	"testing"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
)

func TestRootCommandIncludesResetPasswordCommand(t *testing.T) {
	root := newRootCommand()
	command, _, err := root.Find([]string{"users:reset-password"})
	if err != nil || command == nil {
		t.Fatalf("find users:reset-password: command=%v err=%v", command, err)
	}
}

func TestNormalizeResetPasswordEmail(t *testing.T) {
	got, err := normalizeResetPasswordEmail(" Admin@Example.COM ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "admin@example.com" {
		t.Fatalf("email=%q", got)
	}
	if _, err := normalizeResetPasswordEmail("not an email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestResetPasswordPolicyMessages(t *testing.T) {
	policy := identity.PasswordPolicy{
		MinLength:        12,
		MaxLength:        128,
		RequireLowercase: true,
		RequireUppercase: true,
		RequireNumber:    true,
		RequireSymbol:    true,
	}
	err := validateResetPasswordCandidate(policy, "short")
	if err == nil {
		t.Fatal("expected policy error")
	}
	for _, want := range []string{"密码长度不足", "需要包含大写字母", "需要包含数字", "需要包含符号"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err.Error(), want)
		}
	}
}

func TestRecoverySuperAdminActorCanResetSuperAdmin(t *testing.T) {
	actor := recoverySuperAdminActor()
	if !actor.IsSuperAdmin() || !actor.Can(identity.PermissionUserManage) {
		t.Fatalf("unexpected actor: %+v", actor)
	}
}
