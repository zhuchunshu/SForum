<div v-if="step===4">
    <h3 style="color: red">创建管理员用户</h3>
    <div class="mb-3">
        <label class="form-label">管理员邮箱</label>
        <input v-model="email" type="email" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">管理员用户名</label>
        <input v-model="username" type="text" class="form-control" autocomplete="off" required>
    </div>
    <div class="mb-3">
        <label class="form-label">管理员密码</label>
        <input v-model="password" type="password" class="form-control" autocomplete="off" required>
    </div>
</div>