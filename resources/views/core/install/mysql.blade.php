<div v-if="step===1 && env">
    <div class="mb-3">
        <label class="form-label">数据库地址</label>
        <input v-model="env.DB_HOST" type="text" class="form-control" autocomplete="off"
               required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库名</label>
        <input v-model="env.DB_DATABASE" type="text" class="form-control" autocomplete="off"
               required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库用户名</label>
        <input v-model="env.DB_USERNAME" type="text" class="form-control" autocomplete="off"
               required>
    </div>
    <div class="mb-3">
        <label class="form-label">数据库密码</label>
        <input v-model="env.DB_PASSWORD" type="password" class="form-control" autocomplete="new-password"
               required>
    </div>
</div>