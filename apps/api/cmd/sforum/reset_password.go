package main

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	identity "github.com/zhuchunshu/sforum/apps/api/app/Models/Identity"
	options "github.com/zhuchunshu/sforum/apps/api/app/Models/Options"
	"github.com/zhuchunshu/sforum/apps/api/app/Support/Postgres"
)

type resetPasswordOptions struct {
	DatabaseURL string
}

func newUsersResetPasswordCommand() *cobra.Command {
	opts := resetPasswordOptions{}
	cmd := &cobra.Command{
		Use:     "users:reset-password",
		Aliases: []string{"user:reset-password"},
		Short:   "交互式重置用户密码",
		Long: `交互式重置用户密码。

流程：
  1. 询问邮箱；
  2. 按邮箱查询用户并展示账号摘要；
  3. 二次确认；
  4. 隐藏输入新密码并确认；
  5. 按站点密码策略重置密码、递增 token version，并撤销该用户全部活跃会话。

由于 config.Load() 只读环境变量、不读 .env，运行前需先把 .env 导入环境：

  set -a; . ./.env; set +a
  go run ./cmd/sforum users:reset-password

或用 --database-url 显式覆盖连接串。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUsersResetPasswordCommand(cmd.Context(), opts, cmd)
		},
	}
	cmd.Flags().StringVar(&opts.DatabaseURL, "database-url", "", "PostgreSQL URL (defaults to DATABASE_URL; no other app config is loaded)")
	return cmd
}

func runUsersResetPasswordCommand(ctx context.Context, opts resetPasswordOptions, cmd *cobra.Command) error {
	databaseURL := strings.TrimSpace(opts.DatabaseURL)
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		return fmt.Errorf("database url is empty: set DATABASE_URL or pass --database-url")
	}

	email, err := promptResetPasswordEmail()
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, databaseURL, 2)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	identityStore := identity.NewPostgresStore(pool)
	optionService := options.NewService(options.NewPostgresStore(pool))
	identityService := identity.NewServiceWithPasswordPolicy(identityStore, optionService)

	user, err := identityStore.GetCurrentUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			return fmt.Errorf("user not found for email %q", email)
		}
		return err
	}

	if err := confirmResetPasswordTarget(user); err != nil {
		return err
	}

	policy, err := optionService.PasswordPolicy(ctx)
	if err != nil {
		return fmt.Errorf("load password policy: %w", err)
	}
	password, err := promptResetPasswordNewPassword(policy)
	if err != nil {
		return err
	}

	result, err := identityService.AdminSetUserPassword(ctx, recoverySuperAdminActor(), user.ID, password)
	if err != nil {
		return formatResetPasswordError(err)
	}
	cmd.Printf("密码已重置：user_id=%d username=%s revoked_sessions=%d\n", user.ID, user.Username, result.RevokedSessions)
	return nil
}

func promptResetPasswordEmail() (string, error) {
	var email string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("用户邮箱").
				Description("将使用邮箱精确查询用户。").
				Value(&email).
				Validate(func(value string) error {
					_, err := normalizeResetPasswordEmail(value)
					return err
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return normalizeResetPasswordEmail(email)
}

func confirmResetPasswordTarget(user identity.CurrentUser) error {
	confirmed := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("确认重置这个用户的密码？").
				Description(resetPasswordUserSummary(user)).
				Affirmative("确认重置").
				Negative("取消").
				Value(&confirmed).
				Validate(func(value bool) error {
					if !value {
						return errors.New("已取消重置")
					}
					return nil
				}),
		),
	)
	return form.Run()
}

func promptResetPasswordNewPassword(policy identity.PasswordPolicy) (string, error) {
	var password string
	var confirmation string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("新密码").
				Description(resetPasswordPolicySummary(policy)).
				EchoMode(huh.EchoModePassword).
				Value(&password).
				Validate(func(value string) error {
					return validateResetPasswordCandidate(policy, value)
				}),
			huh.NewInput().
				Title("再次输入新密码").
				EchoMode(huh.EchoModePassword).
				Value(&confirmation).
				Validate(func(value string) error {
					if value != password {
						return errors.New("两次输入的密码不一致")
					}
					return nil
				}),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return password, nil
}

func normalizeResetPasswordEmail(value string) (string, error) {
	email := strings.TrimSpace(value)
	if email == "" {
		return "", errors.New("邮箱不能为空")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", errors.New("请输入有效邮箱地址")
	}
	return strings.ToLower(email), nil
}

func resetPasswordUserSummary(user identity.CurrentUser) string {
	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		displayName = "-"
	}
	return fmt.Sprintf(
		"ID: %d\n用户名: %s\n显示名: %s\n状态: %s\n角色: %s",
		user.ID,
		user.Username,
		displayName,
		user.Status,
		strings.Join(user.RoleKeys, ", "),
	)
}

func resetPasswordPolicySummary(policy identity.PasswordPolicy) string {
	policy = policy.Normalized()
	parts := []string{
		fmt.Sprintf("长度 %d-%d 个字符", policy.MinLength, policy.MaxLength),
	}
	if policy.RequireLowercase {
		parts = append(parts, "包含小写字母")
	}
	if policy.RequireUppercase {
		parts = append(parts, "包含大写字母")
	}
	if policy.RequireNumber {
		parts = append(parts, "包含数字")
	}
	if policy.RequireSymbol {
		parts = append(parts, "包含符号")
	}
	return strings.Join(parts, "；")
}

func validateResetPasswordCandidate(policy identity.PasswordPolicy, password string) error {
	fields := policy.Validate(password)
	if len(fields) == 0 {
		return nil
	}
	return errors.New(resetPasswordFieldMessages(fields))
}

func resetPasswordFieldMessages(fields identity.FieldMessages) string {
	messages := []string{}
	for _, message := range fields[identity.FieldPassword] {
		switch message {
		case identity.MessagePasswordMin:
			messages = append(messages, "密码长度不足")
		case identity.MessagePasswordMax:
			messages = append(messages, "密码长度过长")
		case identity.MessagePasswordLowercase:
			messages = append(messages, "需要包含小写字母")
		case identity.MessagePasswordUppercase:
			messages = append(messages, "需要包含大写字母")
		case identity.MessagePasswordNumber:
			messages = append(messages, "需要包含数字")
		case identity.MessagePasswordSymbol:
			messages = append(messages, "需要包含符号")
		default:
			messages = append(messages, message)
		}
	}
	if len(messages) == 0 {
		return "密码不符合策略"
	}
	return strings.Join(messages, "；")
}

func formatResetPasswordError(err error) error {
	var invalid *identity.RegisterInvalidError
	if errors.As(err, &invalid) {
		return fmt.Errorf("password does not meet policy: %s", resetPasswordFieldMessages(invalid.Fields))
	}
	return err
}

func recoverySuperAdminActor() identity.Actor {
	return identity.Actor{
		ID:       0,
		Status:   identity.UserStatusActive,
		RoleKeys: []string{identity.RoleSuperAdmin},
	}
}
