package options

import "sort"

// OptionGuardManagePermissions 返回给定 option 名称需要的全部管理权限。
// 该方法只读取编译期目录，不读取 Store，供 Route Guard 热路径使用。
// names=nil 表示返回目录中的所有候选权限，用于只读管理列表的 any-of 判定。
func (s *Service) OptionGuardManagePermissions(names []string) ([]string, bool) {
	if s == nil {
		return nil, false
	}
	permissions := map[string]struct{}{}
	if names == nil {
		for _, definition := range optionDefinitions {
			if definition.managePermission != "" {
				permissions[definition.managePermission] = struct{}{}
			}
		}
	} else {
		if len(names) == 0 {
			return nil, false
		}
		for _, name := range names {
			definition, ok := optionDefinitionFor(normalizeName(name))
			if !ok || definition.managePermission == "" {
				return nil, false
			}
			permissions[definition.managePermission] = struct{}{}
		}
	}
	result := make([]string, 0, len(permissions))
	for permission := range permissions {
		result = append(result, permission)
	}
	sort.Strings(result)
	return result, len(result) > 0
}
