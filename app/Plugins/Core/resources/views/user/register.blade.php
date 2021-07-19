<div class="flex flex-col h-screen justify-content-center">
    <div class="hero min-h-screen bg-base-200">
        <div class="card flex-shrink-0 w-full max-w-sm shadow-2xl bg-base-100">
            <div class="card-body" id="vue-core-sign-register">
                <div class="card-title">Register</div>

                <form method="POST" @@submit.prevent="submit">
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Username</span>
                        </label>
                        <input type="text" v-model="username" placeholder="Username" class="input input-bordered">
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Email</span>
                        </label>
                        <input type="email" v-model="email" placeholder="email" class="input input-bordered">
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Password</span>
                        </label>
                        <input type="password" v-model="password" placeholder="password" class="input input-bordered">
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Confirm Password</span>
                        </label>
                        <input type="password" v-model="cfpassword" placeholder="Confirm password" class="input input-bordered">
                        <label class="label">
                            <a href="#" class="label-text-alt">Forgot password?</a>
                        </label>
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Captcha</span>
                            <a href="#" class="label-text-alt">
                               {{plugins_core_captcha()->get()['add1']}}+{{plugins_core_captcha()->get()['add2']}}=?
                            </a>
                        </label>
                        <input placeholder="captcha" v-model="captcha" class="input input-bordered" type="number">
                    </div>
                    <div class="form-control mt-6">
                        <button type="submit" class="btn btn-primary">Register</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>