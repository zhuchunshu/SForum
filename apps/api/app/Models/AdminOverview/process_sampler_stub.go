//go:build !unix

package adminoverview

import "fmt"

// osProcessSampler 非 Unix 平台不提供进程 RSS 采样。
type osProcessSampler struct{}

func (osProcessSampler) List() ([]ProcessSample, error) {
	return nil, fmt.Errorf("process RSS sampling is not supported on this platform")
}

func readSelfRSSFallback() (uint64, bool) {
	return 0, false
}
