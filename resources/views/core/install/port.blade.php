<div v-if="step===4 && env">
    <h3 style="color: red">配置服务端口</h3>
    <div class="mb-3">
        <label class="form-label">WEB服务端口</label>
        <input v-model="env.SERVER_WEB_PORT" type="number" min="0" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">WebSocket服务端口</label>
        <input v-model="env.SERVER_WS_PORT" type="number" min="0" class="form-control" autocomplete="off">
        <small>默认为空</small>
    </div>
    <div class="mb-3">
        <label class="form-label">API服务端口</label>
        <input v-model="env.SERVER_API_PORT" type="number" min="0" class="form-control" autocomplete="off">
    </div>
</div>