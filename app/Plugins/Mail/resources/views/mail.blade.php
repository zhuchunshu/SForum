<div class="card-body">
    <h3 class="card-title">目前只支持SMTP发信</h3>
    <div class="mb-3">
        <label class="form-label">SMTP 主机地址</label>
        <input v-model="env.MAIL_SMTP_HOST" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 端口</label>
        <input v-model="env.MAIL_SMTP_PORT" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 加密方式(tls,ssl)</label>
        <input v-model="env.MAIL_SMTP_ENCRYPTION" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 用户名</label>
        <input v-model="env.MAIL_SMTP_USERNAME" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 密码</label>
        <input v-model="env.MAIL_SMTP_PASSWORD" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 超时时间(秒)</label>
        <input v-model="env.MAIL_SMTP_TIMEOUT" type="text" class="form-control">
    </div>
</div>