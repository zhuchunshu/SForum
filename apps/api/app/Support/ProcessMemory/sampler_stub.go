//go:build !unix

package processmemory

import "fmt"

// osSampler 非 Unix 平台不提供进程 RSS 采样。
type osSampler struct{}

func (osSampler) List() ([]Sample, error) {
	return nil, fmt.Errorf("process RSS sampling is not supported on this platform")
}

// ParsePSList 非 Unix 占位，保持 API 对称。
func ParsePSList([]byte) ([]Sample, error) {
	return nil, fmt.Errorf("process RSS sampling is not supported on this platform")
}

// ReadSelfRSSFallback 非 Unix 无 /proc 回退。
func ReadSelfRSSFallback() (uint64, bool) {
	return 0, false
}
