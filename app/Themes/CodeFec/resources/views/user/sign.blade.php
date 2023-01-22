<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{session()->get('theme','antialiased')}}">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>{{$title}} - {{ get_options('title', config('app_name', 'CodeFec')) }}</title>
    <link rel="stylesheet" href="{{ mix('plugins/Core/css/app.css') }}">
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <script>
        var csrf_token = "{{ csrf_token() }}";
    </script>
    <meta name="description" content="{{ get_options('description') }}">
    <meta name="keywords" content="{{ get_options('keywords') }}">
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-vendors.min.css' }}" rel="stylesheet" />
    <link rel="stylesheet" href="{{mix("plugins/Core/css/core.css")}}">
{{--    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet" />--}}
    <script>
        var csrf_token = "{{ csrf_token() }}";
        var ws_url = "{{ws_url()}}";
        var login_token = "{{auth()->token()}}";
    </script>
    @yield('css')
    @yield('headers')
</head>

<body  class=" border-top-wide border-primary d-flex flex-column">
<div class="page page-center">
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
                        @if(get_options('core_user_otlogin','开启')==='开启' && @$login!==false)
                        <div class="hr-text">or</div>
                        <div class="card-body">
                            <div class="row">
                                <div class="col"><a href="#" class="btn w-100">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/brand-github -->
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon text-github" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 19c-4.3 1.4 -4.3 -2.5 -6 -3m12 5v-3.5c0 -1 .1 -1.4 -.5 -2c2.8 -.3 5.5 -1.4 5.5 -6a4.6 4.6 0 0 0 -1.3 -3.2a4.2 4.2 0 0 0 -.1 -3.2s-1.1 -.3 -3.5 1.3a12.3 12.3 0 0 0 -6.2 0c-2.4 -1.6 -3.5 -1.3 -3.5 -1.3a4.2 4.2 0 0 0 -.1 3.2a4.6 4.6 0 0 0 -1.3 3.2c0 4.6 2.7 5.7 5.5 6c-.6 .6 -.6 1.2 -.5 2v3.5" /></svg>
                                        Login with Github
                                    </a></div>
                                <div class="col"><a href="#" class="btn w-100">
                                        <!-- Download SVG icon from http://tabler-icons.io/i/brand-twitter -->
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon text-twitter" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M22 4.01c-1 .49 -1.98 .689 -3 .99c-1.121 -1.265 -2.783 -1.335 -4.38 -.737s-2.643 2.06 -2.62 3.737v1c-3.245 .083 -6.135 -1.395 -8 -4c0 0 -4.182 7.433 4 11c-1.872 1.247 -3.739 2.088 -6 2c3.308 1.803 6.913 2.423 10.034 1.517c3.58 -1.04 6.522 -3.723 7.651 -7.742a13.84 13.84 0 0 0 .497 -3.753c-.002 -.249 1.51 -2.772 1.818 -4.013z" /></svg>
                                        Login with Twitter
                                    </a></div>
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
                <img src="/plugins/Core/image/undraw_secure_login_pdn4.svg" height="300" class="d-block mx-auto" alt="">
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
</body>
