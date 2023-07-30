<div id="vue-core-sign-register">
    <form @@submit.prevent="submit" autocomplete="off">
        <div class="mb-3">
            <label class="form-label">邮箱</label>
            <input autocomplete="off" type="email" v-model="email" class="form-control" placeholder="Enter email" required>
        </div>
        <div class="mb-3">
            <label class="form-label">用户名</label>
            <input autocomplete="off" type="text" v-model="username" class="form-control" placeholder="Enter username" required>
        </div>
        <div class="mb-3">
            <label class="form-label">
                密码
            </label>
            <input type="password" v-model="password" class="form-control" placeholder="Password" autocomplete="new-password" required>
        </div>
        <div class="mb-2">
            <label class="form-label">
                重复密码
            </label>
            <input type="password" v-model="cfpassword" class="form-control" placeholder="Password" autocomplete="new-password" required>
        </div>

        @if(get_options('core_user_reg_yaoqing','关闭')==='开启')
            <div class="mb-3">
                <label for="" class="form-label">
                    邀请码
                    @if(get_options('core_user_reg_yaoqing_url'))
                        <span class="form-label-description">
                                <a target="_blank" href="{{get_options('core_user_reg_yaoqing_url')}}">获取邀请码</a>
                            </span>
                    @endif
                </label>
                <div class="input-group">
                    <input type="text" v-model="invitationCode" class="form-control" placeholder="invitation Code" autocomplete="off" required>
                </div>
            </div>
        @endif

        @if(get_options('core_user_reg_captcha','开启')==='开启')
            <div class="mb-3">
                <label for="" class="form-label">验证码</label>
                <input type="hidden" isCaptchaInput v-model="captcha" class="form-control" placeholder="captcha" autocomplete="off" required>
                <div id="captcha-container"></div>

            </div>
        @endif

        <div class="form-footer">
            <button type="submit" class="btn btn-primary w-100">立即注册</button>
        </div>
    </form>
</div>