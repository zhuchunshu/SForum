<div id="vue-core-sign-login-username">
    <form  @@submit.prevent="submit">
        <div class="mb-3">
            <label class="form-label">用户名
                <span class="form-label-description">
                  <a href="/login/email?redirect={{request()->input('redirect','/')}}">邮箱登陆</a>
                </span></label>
            <input type="text" v-model="username" class="form-control" placeholder="Enter username" required>
        </div>
        <div class="mb-2">
            <label class="form-label">
                密码
                <span class="form-label-description">
                  <a href="/forgot-password">忘记密码?</a>
                </span>
            </label>
            <input type="password" v-model="password" class="form-control" placeholder="Password" autocomplete="off" required>
        </div>
        @if(get_options('core_user_login_captcha','开启')==='开启')
            <div class="mb-3">
                <label for="" class="form-label">验证码</label>
                <input type="hidden" isCaptchaInput v-model="captcha" class="form-control" placeholder="captcha" autocomplete="off"
                       required>
                <div id="captcha-container"></div>

            </div>
        @endif

        <div class="form-footer">
            <button type="submit" isNeedCaptcha disabled class="btn btn-primary w-100">登陆</button>
        </div>
    </form>
</div>