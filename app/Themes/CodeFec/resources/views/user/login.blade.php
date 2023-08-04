<div id="vue-core-sign-login">
    <form action="./" method="get" autocomplete="off" novalidate @@submit.prevent="submit">
        <div class="mb-3">
            <label class="form-label">邮箱
                <span class="form-label-description">
                  <a href="/login?redirect={{request()->input('redirect','/')}}">用户名登陆</a>
                </span></label>
            <input type="email" v-model="email" class="form-control" placeholder="Enter email" required>
        </div>
        <div class="mb-2">
            <label class="form-label">
                密码
                <span class="form-label-description">
                  <a href="/forgot-password">忘记密码?</a>
                </span>
            </label>
            <input type="password" v-model="password" class="form-control" placeholder="Password" autocomplete="off"
                   required>
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