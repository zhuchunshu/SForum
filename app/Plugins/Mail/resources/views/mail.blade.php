<div class="card-body">
    <h3 class="card-title">目前只支持SMTP发信</h3>
    <span class="text-muted"><a href="/admin/mail">点我测试发信</a></span>
    <div class="mb-3">
        <label class="form-label">SMTP 主机地址</label>
        <input v-model="data.MAIL_SMTP_HOST" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 端口</label>
        <input v-model="data.MAIL_SMTP_PORT" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">发信邮箱</label>
        <input v-model="data.MAIL_SMTP_FORM_MAIL" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">发信名称</label>
        <input v-model="data.MAIL_SMTP_FORM_NAME" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 用户名</label>
        <input v-model="data.MAIL_SMTP_USERNAME" type="text" class="form-control">
    </div>
    <div class="mb-3">
        <label class="form-label">SMTP 密码</label>
        <input v-model="data.MAIL_SMTP_PASSWORD" type="text" class="form-control">
    </div>
</div>