<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{session()->get('theme','antialiased')}}">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>{{$title}} - {{ get_options('title', config('app_name', 'CodeFec')) }}</title>
    <link rel="stylesheet" href="{{ mix('plugins/Core/css/app.css') }}">
    <meta name="description" content="{{ get_options('description') }}">
    <meta name="keywords" content="{{ get_options('keywords') }}">
    <link rel="icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link rel="shortcut icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-vendors.min.css' }}" rel="stylesheet" />
    <link rel="stylesheet" href="{{mix("plugins/Core/css/core.css")}}">
{{--    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet" />--}}
    <script>
        const csrf_token = "{{ csrf_token() }}";
        var theme_status = @if(session()->has('theme')) {{"true"}} @else {{"false"}} @endif;
       const captcha_config = {
            cloudflare: "{{get_options("admin_captcha_cloudflare_turnstile_website_key","1x00000000000000000000AA")}}",
            recaptcha: "{{get_options("admin_captcha_recaptcha_website_key")}}",
            service:"{{get_options("admin_captcha_service")}}"
        }
        const system_theme = "{{session()->get('theme',session()->get('auto_theme','light'))}}"
        var auto_theme = "{{session()->get('auto_theme','light')}}";
    </script>
    @yield('css')
    @yield('headers')
</head>

<body data-bs-theme="{{session()->get('theme',session()->get('auto_theme','light'))}}" class="border-top-wide border-primary d-flex flex-column">
<div class="page page-center">
    @include("App::layouts.errors")
    @include("App::layouts._msg")
    <div class="container container-normal py-4">
        <div class="row align-items-center g-4">
            <div class="col-lg">
                <div class="container-tight">
                    <div class="text-center mb-4">
                        @if(!get_options('web_logo')) <a class="navbar-brand navbar-brand-autodark" href="/">{{get_options('web_name', 'CodeFec')}}</a>@else
                            @include(get_component_view_name(get_options('web_logo'))) @endif
                    </div>
                    <div class="card card-md">
                        <div class="card-body">
                            @if(@$login!==false)<h2 class="h2 text-center mb-4">登录到您的帐户</h2>@endif
                            @if(@$register===true)<h2 class="h2 text-center mb-4">注册新的账号</h2>@endif
                            @include($view)
                        </div>
                        @if(get_options('core_user_otlogin','开启')==='开启' && @$login!==false && count((new \App\Plugins\User\src\Service\Oauth2())->get_enables()))
                        <div class="hr-text">or</div>
                        <div class="card-body">
                            <div class="row">
{{--                                <div class="col"><a href="#" class="btn w-100">--}}
{{--                                        <!-- Download SVG icon from http://tabler-icons.io/i/brand-github -->--}}
{{--                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon text-github" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" /></svg>--}}
{{--                                        Login with Github--}}
{{--                                    </a></div>--}}
                                @foreach((new \App\Plugins\User\src\Service\Oauth2())->get_enables_data() as $data)
                                    @include($data['view'])
                                @endforeach
                            </div>
                        </div>
                        @endif
                    </div>
                    @if(@$register!==true)
                        <div class="text-center text-muted mt-3">
                            还没有账号? <a href="/register" tabindex="-1">立即注册</a>
                        </div>
                    @endif
                    @if(@$register===true)
                        <div class="text-center text-muted mt-3">
                            已有账号? <a href="/login" tabindex="-1">立即登陆</a>
                        </div>
                    @endif
                </div>
            </div>
            <div class="col-lg d-none d-lg-block">
                <img src="{{get_options('sign_page_image','/plugins/Core/image/undraw_secure_login_pdn4.svg')}}" height="300" class="d-block mx-auto" alt="">
            </div>
        </div>
    </div>
</div>

<script src='/js/jquery-3.6.0.min.js'></script>
<script src="{{ mix('js/vue.js') }}"></script>
<script src="{{ '/tabler/js/tabler.min.js' }}"></script>
<script src="{{ mix('plugins/Core/js/app.js') }}"></script>
{{--<script src="{{ file_hash('js/diy.js') }}"></script>--}}
{{-- <!-- 自定义Js --> --}}
@foreach (\App\CodeFec\Ui\functions::get('js') as $key => $value)
    <script src="{{ $value }}"></script>
@endforeach
{{--插件js--}}
@foreach((new \App\CodeFec\Plugins())->getEnPlugins() as $value)
    @if(file_exists(public_path("plugins/".$value."/".$value.".js")))
        <script src="{{ file_hash("plugins/".$value."/".$value.".js") }}"></script>
    @endif
@endforeach
@yield('scripts')
<script>
    var redirect_url = "{{request()->input('redirect','/')}}"
</script>
<script src="{{ mix('plugins/Core/js/sign.js') }}"></script>
@if(get_options("admin_captcha_service","cloudflare")==="google")
    <script src="//www.recaptcha.net/recaptcha/api.js?onload=onloadGoogleRecaptchaCallback" async
            defer></script>
@else
    <script src="//challenges.cloudflare.com/turnstile/v0/api.js?onload=onloadTurnstileCallback" async
            defer></script>
@endif
</body>
