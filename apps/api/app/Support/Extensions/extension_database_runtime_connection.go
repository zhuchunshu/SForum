package extensionsruntime

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func extensionDatabaseRuntimeConnectionURL(
	base *pgx.ConnConfig,
	credential ExtensionDatabaseRuntimeCredential,
) (string, error) {
	if base == nil || !validPostgresIdentifier(credential.RoleName) ||
		!validPostgresCatalogName(credential.DatabaseName) ||
		!extensionDatabasePasswordPattern.MatchString(credential.Password) {
		return "", ErrExtensionDatabaseRegistryInvalid
	}
	raw := strings.TrimSpace(base.ConnString())
	var connectionURL *url.URL
	if strings.HasPrefix(raw, "postgres://") || strings.HasPrefix(raw, "postgresql://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" {
			return "", ErrExtensionDatabaseCredential
		}
		connectionURL = parsed
	} else {
		connectionURL = &url.URL{Scheme: "postgres"}
		query := connectionURL.Query()
		if strings.HasPrefix(base.Host, "/") {
			query.Set("host", base.Host)
			query.Set("port", strconv.FormatUint(uint64(base.Port), 10))
		} else {
			connectionURL.Host = net.JoinHostPort(base.Host, strconv.FormatUint(uint64(base.Port), 10))
		}
		switch {
		case base.TLSConfig == nil:
			query.Set("sslmode", "disable")
		case base.TLSConfig.InsecureSkipVerify:
			query.Set("sslmode", "require")
		default:
			query.Set("sslmode", "verify-full")
		}
		if base.ConnectTimeout > 0 {
			query.Set("connect_timeout", strconv.Itoa(int(base.ConnectTimeout.Seconds())))
		}
		connectionURL.RawQuery = query.Encode()
	}

	connectionURL.User = url.UserPassword(credential.RoleName, credential.Password)
	connectionURL.Path = "/" + credential.DatabaseName
	connectionURL.RawPath = ""
	query := connectionURL.Query()
	for _, key := range []string{
		"user", "password", "dbname", "database", "role", "search_path", "options",
		"pool_max_conns", "pool_min_conns", "pool_max_conn_lifetime",
		"pool_max_conn_idle_time", "pool_health_check_period",
	} {
		query.Del(key)
	}
	query.Set("application_name", extensionDatabaseRuntimeApplicationName(credential))
	connectionURL.RawQuery = query.Encode()
	result := connectionURL.String()
	parsed, err := pgx.ParseConfig(result)
	if err != nil || parsed.User != credential.RoleName || parsed.Password != credential.Password ||
		parsed.Database != credential.DatabaseName {
		return "", fmt.Errorf("%w: validate runtime connection URL", ErrExtensionDatabaseCredential)
	}
	return result, nil
}

func extensionDatabaseRuntimeApplicationName(credential ExtensionDatabaseRuntimeCredential) string {
	leaseID := credential.LeaseID
	if len(leaseID) > 12 {
		leaseID = leaseID[:12]
	}
	return "sforum-plugin:" + credential.Artifact.ExtensionID + ":" + leaseID
}
