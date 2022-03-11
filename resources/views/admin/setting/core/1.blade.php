<div class="card-body">
    <div class="mb-3">
        <label class="form-label">应用名称</label>
        <input type="text" min="1" max="3" class="form-control" v-model="data.APP_NAME">
        <small>当前: {{get_options('APP_NAME',env('APP_NAME','未配置'))}}</small>
    </div>
    <div class="mb-3">
        <label class="form-label">更新加速
        </label>
        <select class="form-select" v-model="data.update_server">
            <option value="1">大陆服务器加速</option>
            <option value="2">境外加速</option>
        </select>
        <small>默认境外</small>
    </div>
    <div class="mb-3">
        <label class="form-label">网站名称</label>
        <input type="text" min="1" max="3" class="form-control" v-model="data.web_name">
    </div>
    <div class="mb-3">
        <label class="form-label">网站标题</label>
        <input v-model="data.title" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">网站首页标题</label>
        <input v-model="data.home_title" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">网站地址</label>
        <input v-model="data.APP_URL" type="text" class="form-control">
        <small>当前:{{url()}}</small>
    </div>
    <div class="mb-3">
        <label class="form-label">网站websocket地址</label>
        <input v-model="data.APP_WS_URL" type="text" class="form-control">
        <small>当前:{{ws_url()}}</small>
    </div>
    <div class="mb-3">
        <label class="form-label">网站关键字</label>
        <input v-model="data.keywords" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">网站描述</label>
        <input v-model="data.description" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">网站备案号</label>
        <input v-model="data.icp" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">公安备案号</label>
        <input v-model="data.ga_icp" type="text" class="form-control">
    </div>
</div>