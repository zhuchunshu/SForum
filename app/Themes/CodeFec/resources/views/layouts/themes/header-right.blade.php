<div class="nav-item d-none d-md-flex me-3">
    <div class="btn-list">
        {{--                        <a href="https://github.com/tabler/tabler" class="btn btn-outline-white" target="_blank" rel="noreferrer">--}}
        {{--                            <!-- Download SVG icon from http://tabler-icons.io/i/brand-github -->--}}
        {{--                            <svg xmlns="http://www.w3.org/2000/svg" class="icon text-github" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" /></svg>--}}
        {{--                            Source code--}}
        {{--                        </a>--}}
    </div>
</div>

<div class="nav-item dropdown me-3 d-none d-md-inline-flex">
    @if(session()->get('theme','theme-dark')!=="theme-dark")
        <a href="#" id="core_update_theme" class="px-0 nav-link">
            <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-moon" width="24" height="24"
                 viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                 stroke-linejoin="round">
                <desc>Download more icon variants from https://tabler-icons.io/i/moon</desc>
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <path d="M12 3c.132 0 .263 0 .393 0a7.5 7.5 0 0 0 7.92 12.446a9 9 0 1 1 -8.313 -12.454z"></path>
            </svg>
        </a>
    @else
        <a href="#" id="core_update_theme" class="px-0 nav-link">
            <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
            <svg class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                 fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path d="M0 0h24v24H0z" stroke="none"></path>
                <circle cx="12" cy="12" r="4"></circle>
                <path d="M3 12h1m8-9v1m8 8h1m-9 8v1M5.6 5.6l.7.7m12.1-.7l-.7.7m0 11.4l.7.7m-12.1-.7l-.7.7"></path>
            </svg>
        </a>
    @endif
</div>
@if(auth()->check())

    <div class="nav-item d-none d-md-flex me-3">
        <a href="/user/notice" class="px-0 nav-link">
            <!-- Download SVG icon from http://tabler-icons.io/i/bell -->
            <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24"
                 stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                <path d="M10 5a2 2 0 0 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6"/>
                <path d="M9 17v1a3 3 0 0 0 6 0v-1"/>
            </svg>
            <span id="core-notice-red" class="badge bg-red badge-blink" style="display: none;    position: absolute!important;
    right: 0!important;
    transform: translate(50%,-50%);
    z-index: 1;"></span>
        </a>
    </div>


    <div id="vue-header-right-my" class="nav-item dropdown">
        <a href="#" class="p-0 nav-link d-flex lh-1 text-reset" data-bs-toggle="dropdown" aria-label="Open user menu">
            <span class="avatar avatar-sm avatar-rounded" style="background-image: url({{super_avatar(auth()->data())}})"></span>
            <span id="common-user-notice-2" class="badge bg-red badge-blink d-md-none" style="transform: translate(50%, -50%);display: none; top: -2px; z-index: 1; position: absolute !important; right: 0px !important;"></span>
            <div class="d-none d-xl-block ps-2">
                <div>{{auth()->data()->username}}</div>
                <div class="mt-1 small text-muted">{{__("user.st member",['member' => auth()->id()])}}</div>
            </div>
        </a>
        <div class="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
            <a href="/user" class="dropdown-item">个人中心</a>
            <a href="/user/collections" class="dropdown-item">我的收藏</a>
            <a href="/user/notice" class="dropdown-item">
{{--                <!-- Download SVG icon from http://tabler-icons.io/i/bell -->--}}
{{--                --}}{{--                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 5a2 2 0 0 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6" /><path d="M9 17v1a3 3 0 0 0 6 0v-1" /></svg>--}}
                我的通知
                <span id="common-user-notice-1" class="badge bg-red ms-2" style="display: none"></span>
            </a>
            <div class="dropdown-divider"></div>
            <a href="/user/setting" class="dropdown-item">个人设置</a>
            <a href="#" @@click="Logout" class="dropdown-item">退出</a>
        </div>
    </div>
@else
    <div class="nav-item dropdown">
        <span class="d-none d-md-block">
            <a href="/register" class="btn btn-light">注册</a>
            <a href="/login" class="btn btn-dark">登陆</a>
        </span>
        <span class="d-block d-md-none">
            <a class="px-0 nav-link" href="/login">
<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-circle" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="12" cy="12" r="9"></circle>
   <circle cx="12" cy="10" r="3"></circle>
   <path d="M6.168 18.849a4 4 0 0 1 3.832 -2.849h4a4 4 0 0 1 3.834 2.855"></path>
</svg>
            </a>
        </span>
    </div>
@endif
