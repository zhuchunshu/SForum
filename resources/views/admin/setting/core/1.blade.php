<div class="card-body">
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
        <input v-model="env.APP_URL" type="text" class="form-control">
        <small>默认{{url()}}</small>
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
        <label class="form-label">设置项缓存时间/(秒)</label>
        <input v-model="data.set_cache_time" type="number" class="form-control">
        <small>默认10分钟(600秒),不开启请填写0</small>
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