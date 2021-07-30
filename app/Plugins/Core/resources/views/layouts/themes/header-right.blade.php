
<div class="nav-item d-none d-md-flex me-3">
    <div class="btn-list">
{{--                        <a href="https://github.com/tabler/tabler" class="btn btn-outline-white" target="_blank" rel="noreferrer">--}}
{{--                            <!-- Download SVG icon from http://tabler-icons.io/i/brand-github -->--}}
{{--                            <svg xmlns="http://www.w3.org/2000/svg" class="icon text-github" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" /></svg>--}}
{{--                            Source code--}}
{{--                        </a>--}}
    </div>
</div>
{{--<div class="nav-item dropdown d-none d-md-flex me-3">--}}
{{--    <a href="#" class="px-0 nav-link" data-bs-toggle="dropdown" tabindex="-1" aria-label="Show notifications">--}}
{{--        <!-- Download SVG icon from http://tabler-icons.io/i/bell -->--}}
{{--        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 5a2 2 0 0 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6" /><path d="M9 17v1a3 3 0 0 0 6 0v-1" /></svg>--}}
{{--        <span class="badge bg-red"></span>--}}
{{--    </a>--}}
{{--    <div class="dropdown-menu dropdown-menu-end dropdown-menu-card">--}}
{{--        <div class="card">--}}
{{--            <div class="card-body">--}}
{{--                Lorem ipsum dolor sit amet, consectetur adipisicing elit. Accusamus ad amet consectetur exercitationem fugiat in ipsa ipsum, natus odio quidem quod repudiandae sapiente. Amet debitis et magni maxime necessitatibus ullam.--}}
{{--            </div>--}}
{{--        </div>--}}
{{--    </div>--}}
{{--</div>--}}
@if(auth()->check())
    <div id="vue-header-right-my" class="nav-item dropdown">
        <a href="#" class="p-0 nav-link d-flex lh-1 text-reset" data-bs-toggle="dropdown" aria-label="Open user menu">
            {!! avatar(auth()->data()->id,"avatar-sm") !!}
            <div class="d-none d-xl-block ps-2">
                <div>{{auth()->data()['username']}}</div>
                <div class="mt-1 small text-muted">本站第{{auth()->data()->id}}位会员</div>
            </div>
        </a>
        <div class="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
            <a href="/user" class="dropdown-item">个人中心</a>
            <a href="/user/collections" class="dropdown-item">我的收藏</a>
            <div class="dropdown-divider"></div>
            <a href="/user/setting" class="dropdown-item">个人设置</a>
            <a href="#" @@click="Logout" class="dropdown-item">Logout</a>
        </div>
    </div>
@else
    <div class="nav-item dropdown">
        <span>
            <a href="/register" class="btn btn-light">注册</a>
            <a href="/login" class="btn btn-dark">登陆</a>
        </span>
    </div>
@endif
