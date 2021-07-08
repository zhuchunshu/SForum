<div id="vue-im-form">
    <form @@submit.prevent="submit">
        <div class="mb-3">
            {{-- <div class="form-label">Toggle switches</div> --}}
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="check_username">
                <span class="form-check-label">修改用户名</span>
            </label>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="check_email">
                <span class="form-check-label">修改邮箱</span>
            </label>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="check_password">
                <span class="form-check-label">修改密码</span>
            </label>
        </div>
        <div class="mb-3" v-if="check_username">
            <label class="form-label">用户名</label>
            <input type="text" v-model="username" class="form-control" name="username">
        </div>
        <div class="mb-3" v-if="check_email">
            <label class="form-label">邮箱</label>
            <input type="email" v-model="email" class="form-control" name="email">
        </div>
        <div class="mb-3" v-if="check_password">
            <label class="form-label">原密码</label>
            <input type="password" v-model="old_pwd" class="form-control" name="oldpwd">
        </div>
        <div class="mb-3" v-if="check_password">
            <label class="form-label">新密码</label>
            <input type="password" v-model="new_pwd" class="form-control" name="newpwd">
        </div>
        <button v-if="submit" type="submit" class="btn btn-primary">提交</button>
    </form>
</div>
