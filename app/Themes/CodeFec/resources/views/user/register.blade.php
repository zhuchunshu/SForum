<div id="vue-core-sign-register">
    <div class="text-center mb-4">
        <a href="." class="navbar-brand navbar-brand-autodark">{{get_options("web_name")}}</a>
    </div>
    <form class="card card-md" @@submit.prevent="submit" autocomplete="off">
        <div class="card-body">
            <h2 class="card-title text-center mb-4">注册新用户</h2>
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
                    <div class="input-group">
                        <input type="text" v-model="captcha" class="form-control" placeholder="captcha" autocomplete="off" required>
                        <span class="input-group-link">
                        <img src="{{captcha()->inline()}}" alt="" onclick="this.src='/captcha?id='+Math.random()">
                    </span>
                    </div>
                </div>
            @endif

            <div class="form-footer">
                <button type="submit" class="btn btn-primary w-100">立即注册</button>
            </div>
        </div>
    </form>
    <div class="text-center text-muted mt-3">
        已有账号? <a href="/login" tabindex="-1">立即登陆</a>
    </div>

</div>