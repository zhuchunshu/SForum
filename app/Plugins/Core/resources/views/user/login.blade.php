<div class="flex flex-col h-screen justify-content-center">
    <div class="hero min-h-screen bg-base-200">
        <div class="card flex-shrink-0 w-full max-w-sm shadow-2xl bg-base-100">
            <div class="card-body" id="vue-core-sign-login">
                <div class="card-title">Login</div>

                <form method="POST" @@submit.prevent="submit">
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Email</span>
                        </label>
                        <input type="email" v-model="email" placeholder="email" class="input input-bordered" required>
                    </div>
                    <div class="form-control">
                        <label class="label">
                            <span class="label-text">Password</span>
                        </label>
                        <input type="password" v-model="password" placeholder="password" class="input input-bordered" required>
                    </div>

                    <div class="form-control mt-6">
                        <button type="submit" class="btn btn-primary">Login</button>
                    </div>
                    <div class="divider">No account yet?</div>
                    <div class="form-control mt-6">
                        <a href="/register" class="btn">Sign up</a>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>