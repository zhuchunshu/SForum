package webreleaseruntime

import (
	"sort"
	"strings"
)

var installEnvironmentNames = map[string]struct{}{
	"PATH": {}, "HOME": {}, "TMPDIR": {}, "TMP": {}, "TEMP": {},
	"HTTP_PROXY": {}, "HTTPS_PROXY": {}, "ALL_PROXY": {}, "NO_PROXY": {},
	"http_proxy": {}, "https_proxy": {}, "all_proxy": {}, "no_proxy": {},
	"BUN_CONFIG_REGISTRY": {}, "NPM_CONFIG_REGISTRY": {},
}

var buildEnvironmentNames = map[string]struct{}{
	"APP_NAME": {}, "APP_URL": {}, "APP_LOCALE": {}, "SUPPORTED_LOCALES": {},
	"NUXT_PUBLIC_API_BASE_URL": {}, "NUXT_PUBLIC_ADMIN_ROUTE_PREFIX": {},
	"NUXT_PUBLIC_I18N_BASE_URL": {},
}

var buildOverrideNames = map[string]struct{}{
	"NUXT_BUILD_DIR": {}, "SFORUM_NITRO_OUTPUT_DIR": {}, "SFORUM_THEME_LAYER": {},
	"SFORUM_ADMIN_REGISTRY_ROOT": {}, "SFORUM_WEB_RELEASE_ID": {},
	"HOST": {}, "PORT": {},
}

func InstallEnvironment(source []string) []string {
	return filterEnvironment(source, installEnvironmentNames, nil)
}

func BuildEnvironment(source []string, overrides map[string]string) []string {
	allowed := make(map[string]struct{}, len(installEnvironmentNames)+len(buildEnvironmentNames))
	for name := range installEnvironmentNames {
		allowed[name] = struct{}{}
	}
	for name := range buildEnvironmentNames {
		allowed[name] = struct{}{}
	}
	return filterEnvironment(source, allowed, overrides)
}

func filterEnvironment(source []string, allowed map[string]struct{}, overrides map[string]string) []string {
	values := make(map[string]string)
	for _, item := range source {
		name, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, accepted := allowed[name]; accepted {
			values[name] = value
		}
	}
	for name, value := range overrides {
		if _, accepted := buildOverrideNames[name]; accepted {
			values[name] = value
		}
	}
	result := make([]string, 0, len(values))
	for name, value := range values {
		result = append(result, name+"="+value)
	}
	sort.Strings(result)
	return result
}
