<style>
    .slide-fade-enter-active {
        transition: all .1s ease-out;
    }

    .slide-fade-leave-active {
        transition: all .2s cubic-bezier(1.0, 0.5, 0.8, 1.0);
    }

    .slide-fade-enter-from,
    .slide-fade-leave-to {
        transform: translateX(20px);
        opacity: 0;
    }
</style>
<div id="vue-core-forgot-password">
    <div class="text-center mb-4">
        <a href="." class="navbar-brand navbar-brand-autodark">{{get_options("web_name")}}</a>
    </div>
    <transition name="slide-fade">
        <transition v-if="sendCoded" name="slide-fade">
            <div v-if="show">
                <form class="card card-md" @@submit.prevent="sendCode">
                    <div class="card-body">
                        <h2 class="card-title text-center mb-4">找回密码</h2>

                        <div class="mb-3">
                            <label class="form-label">邮箱
                                <span class="form-label-description">
                </span></label>
                            <input type="email" v-model="email" class="form-control" placeholder="Enter email" required>
                        </div>
                        <div class="mb-3">
                            <label for="" class="form-label">验证码</label>
                            <div class="input-group">
                                <input type="text" v-model="captcha" class="form-control" placeholder="captcha"
                                       autocomplete="off" required>
                                <span class="input-group-link">
                        <img class="captcha" src="{{captcha()->inline()}}" alt="" onclick="this.src='/captcha?id='+Math.random()">
                    </span>
                            </div>
                        </div>

                        <div class="form-footer">
                            <button type="submit" class="btn btn-primary w-100">发送6位邮箱验证码</button>
                        </div>
                    </div>
                </form>
            </div>
            <div v-else>
                <form class="card card-md" @@submit.prevent="submit">
                    <div class="card-body">
                        <h2 class="card-title text-center mb-4">找回密码</h2>
                        <div class="mb-3">
                            <label for="" class="form-label">邮箱验证码</label>
                            <input type="text" v-model="YzCode" class="form-control" autocomplete="off" required>
                        </div>

                        <div class="form-footer">
                            <button type="submit" class="btn btn-primary w-100">提交</button>
                        </div>
                    </div>
                </form>
            </div>
        </transition>

        <transition v-else name="slide-fade">
            <div v-if="setpwd">
                <form class="card card-md" @@submit.prevent="setPwdSubmit">
                    <div class="card-body">
                        <h2 class="card-title text-center mb-4">设置新密码</h2>

                        <div class="mb-3">
                            <label class="form-label">新密码</label>
                            <input type="password" v-model="setPwd_password" class="form-control"
                                   placeholder="Enter password" required>
                        </div>

                        <div class="mb-3">
                            <label class="form-label">确认密码</label>
                            <input type="password" v-model="setPwd_cfpassword" class="form-control"
                                   placeholder="Enter password" required>
                        </div>


                        <div class="form-footer">
                            <button type="submit" class="btn btn-primary w-100">确认修改</button>
                        </div>
                    </div>
                </form>
            </div>
            <div v-else>
                <div class="empty">
                    <div class="empty-icon">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-mood-happy"
                             width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                             fill="none" stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <circle cx="12" cy="12" r="9"></circle>
                            <line x1="9" y1="9" x2="9.01" y2="9"></line>
                            <line x1="15" y1="9" x2="15.01" y2="9"></line>
                            <path d="M8 13a4 4 0 1 0 8 0m0 0h-8"></path>
                        </svg>
                    </div>
                    <p class="empty-title">密码修改成功</p>
                    <p class="empty-subtitle text-muted">
                        以为您自动登陆,点击下方按钮返回首页
                    </p>
                    <div class="empty-action">
                        <a href="/" class="btn btn-primary">
                            返回首页
                        </a>
                    </div>
                </div>
            </div>
        </transition>
    </transition>
    <div class="text-center text-muted mt-3">
        想起来了? <a href="/login" tabindex="-1">立即登陆</a>
    </div>

</div>