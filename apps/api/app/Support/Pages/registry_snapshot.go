package pages

// 贡献快照与诊断辅助：与注册/绑定逻辑分离，registry.go 因此保持在基线以下
// （见 tests/architecture-boundaries-baseline.json）。行为与拆分前逐字节等价。

// Snapshot 返回当前贡献的只读拷贝（测试/诊断）。
func (r *Registry) Snapshot() (replace map[string][]PageContribution, added map[string]PageContribution) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	replace = make(map[string][]PageContribution, len(r.contributions))
	for k, v := range r.contributions {
		replace[k] = append([]PageContribution(nil), v...)
	}
	added = make(map[string]PageContribution, len(r.compiledAdds))
	for _, cr := range r.compiledAdds {
		added[cr.Pattern] = cr.Contribution
	}
	return replace, added
}
