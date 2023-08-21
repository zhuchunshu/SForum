<div class="card card-body">
    <div class="row">
        <div class="col-lg-4 mb-3">
            <label class="form-label">更新加速
            </label>
            <select class="form-select" v-model="data.update_server">
                <option value="1">大陆服务器加速</option>
                <option value="2">境外加速</option>
            </select>
            <small>默认境外</small>
        </div>

        <div class="col-lg-4 mb-3">
            <label class="form-label">后台主题 - 页头
            </label>
            <select class="form-select" v-model="data.admin_theme_header">
                <option value="1">白</option>
                <option value="2">黑</option>
                <option value="3">透明</option>
            </select>
            <small>默认:白</small>
        </div>
        <div class="col-lg-4 mb-3">
            <label class="form-label">后台主题 - 菜单
            </label>
            <select class="form-select" v-model="data.admin_theme_menu">
                <option value="1">白</option>
                <option value="2">黑</option>
                <option value="3">透明</option>
            </select>
            <small>默认:白</small>
        </div>

        <div class="col-lg-4 mb-3">
            <div class="form-label">SForum通知</div>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.sforum_notice">
                <span class="form-check-label">关闭</span>
            </label>
        </div>

        <div class="col-lg-4 mb-3">
            <div class="form-label">Github API 地址</div>
            <input v-model="data.github_api_url" placeholder="Github API 地址" type="text" class="form-control">
            <small class="text-muted">默认是 https://api.github.com </small>
        </div>

    </div>
</div>