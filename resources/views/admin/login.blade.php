<!doctype html>
<html lang="zh-CN">

<head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"/>
    <meta http-equiv="X-UA-Compatible" content="ie=edge"/>
    <title>Sign in - {{get_options('title', config('app_name', 'CodeFec')) }}</title>
    <meta name="msapplication-TileColor" content="#206bc4"/>
    <meta name="theme-color" content="#206bc4"/>
    <link rel="icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link rel="shortcut icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <!-- CSS files -->
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet"/>
    <link rel="stylesheet" href="{{ mix('iziToast/css/iziToast.min.css') }}">
    <script src="{{ mix('iziToast/js/iziToast.min.js') }}"></script>
    <script>
        var theme_status = @if(session()->has('theme')) {{"true"}} @else {{"false"}} @endif;
        const captcha_config = {
            cloudflare: "{{get_options("admin_captcha_cloudflare_turnstile_website_key","1x00000000000000000000AA")}}",
            recaptcha: "{{get_options("admin_captcha_recaptcha_website_key")}}",
            service: "{{get_options("admin_captcha_service")}}"
        }
        const system_theme = "{{session()->get('theme',session()->get('auto_theme','light'))}}"
        var auto_theme = "{{session()->get('auto_theme','light')}}";
        var csrf_token = "{{csrf_token()}}";</script>
</head>

<body data-bs-theme="{{session()->get('theme',session()->get('auto_theme','light'))}}"
      class="antialiased border-top-wide border-primary d-flex flex-column">
@include("layouts.errors")
@include("layouts._msg")
<div class="page page-center">
    <div class="container-tight py-4">
        <div class="text-center mb-4">
            <a href="#"><img src="/logo.svg" height="36" alt=""></a>
        </div>
        <div id="form">
            <form @@submit.prevent="submit" class="card card-md" action="/admin/login" method="post"
                  autocomplete="off">
                <div class="card-body">
                    <h2 class="card-title text-center mb-4">登陆</h2>
                    <div class="mb-3">
                        <label class="form-label">Username</label>
                        <input type="text" v-model="username" class="form-control" placeholder="Enter Username">
                    </div>
                    <div class="mb-2">
                        <label class="form-label">
                            Password
                        </label>
                        <div class="input-group input-group-flat">
                            <input type="password" v-model="password" class="form-control" placeholder="Password"
                                   autocomplete="off">
                        </div>
                    </div>
                    @if(get_options('admin_login_captcha_off')!=='true')
                        <div class="mb-1">
                            <label for="" class="form-label">验证码</label>
                            <input isCaptchaInput type="hidden" v-model="captcha" class="form-control"
                                   placeholder="captcha"
                                   autocomplete="off" required>
                            <div id="captcha-container"></div>
                        </div>
                    @endif
                    <div class="form-footer" id="submit">
                        <button @if(get_options('admin_login_captcha_off')!=='true') isNeedCaptcha disabled
                                @endif type="submit" class="btn btn-primary w-100">登录
                        </button>
                    </div>
                </div>
            </form>
        </div>
    </div>
</div>
<script src="/js/jquery-3.6.0.min.js"></script>
<script src="{{ mix('js/vue.js') }}"></script>
<script src="{{ mix('js/app.js') }}"></script>
<script src="{{ '/tabler/libs/apexcharts/dist/apexcharts.min.js' }}"></script>
<!-- Tabler Core -->
<script src="{{ '/tabler/js/tabler.min.js' }}"></script>
<script type="module" src="{{ mix('js/admin/login.js') }}"></script>
@if(get_options('admin_login_captcha_off')!=='true')
    @if(get_options("admin_captcha_service","cloudflare")==="google")
        <script src="//www.recaptcha.net/recaptcha/api.js?onload=onloadGoogleRecaptchaCallback" async
                defer></script>
    @else
        <script src="//challenges.cloudflare.com/turnstile/v0/api.js?onload=onloadTurnstileCallback" async
                defer></script>
    @endif
@endif
</body>

</html>
