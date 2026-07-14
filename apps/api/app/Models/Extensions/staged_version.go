package extensions

// StagedArtifact 返回等待可信生命周期执行的不可变候选制品。
// 返回值不携带当前运行时或下一候选，避免调用方误把活动实例状态归到尚未启动的代码上。
func (e Extension) StagedArtifact() (Extension, bool) {
	if e.StagedVersion == nil {
		return Extension{}, false
	}
	staged := e.StagedVersion
	return Extension{
		ID: e.ID, Name: e.Name, Version: staged.Version, Type: e.Type, Status: e.Status,
		Source: e.Source, IsSystem: e.IsSystem, IsDeletable: e.IsDeletable,
		Manifest: staged.Manifest, PackageDigest: staged.PackageDigest,
		AdminFrontendDigest: staged.AdminFrontendDigest, PackagePath: staged.PackagePath,
		ActiveVersionID: staged.ID, InstalledAt: staged.InstalledAt, UpdatedAt: e.UpdatedAt,
	}, true
}

func trustReviewArtifact(extension Extension) Extension {
	// A disabled plugin restarts its current exact artifact. Its staged package
	// stays inert until an explicit upgrade, so the challenge must bind current.
	if staged, ok := extension.StagedArtifact(); ok && extension.Status != StatusDisabled {
		return staged
	}
	return extension
}
