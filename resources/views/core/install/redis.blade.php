<div v-if="step===2 && env">
    <h3 style="color: red">配置Redis</h3>
    <div class="mb-3">
        <label class="form-label">Redis地址</label>
        <input v-model="env.REDIS_HOST" type="text" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">Redis密码</label>
        <input v-model="env.REDIS_AUTH" type="text" class="form-control" autocomplete="off">
        <small>默认为空</small>
    </div>
    <div class="mb-3">
        <label class="form-label">Redis端口</label>
        <input v-model="env.REDIS_PORT" type="text" class="form-control" autocomplete="off">
    </div>
</div>