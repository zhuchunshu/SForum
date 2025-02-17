<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{session()->get('theme','antialiased')}}">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">

    <title>@if(request()->path()==="/")
            @if(get_options('home_title'))
                @yield("title") {{get_options('home_title')}}
            @else
                @yield("title","标题") - {{ get_options('title', config('app_name', 'CodeFec')) }}
            @endif
        @else
            @yield("title","标题") - {{ get_options('title', config('app_name', 'CodeFec')) }}
        @endif</title>
    <link rel="icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link rel="shortcut icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet"/>
    <link rel="stylesheet" href="{{mix("plugins/Core/css/core.css")}}">
    {{--    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet" />--}}
    <link rel="stylesheet" href="{{mix('css/app.css')}}">
    <script>
        var theme_status = @if(session()->has('theme')) {{"true"}} @else {{"false"}} @endif;
        const captcha_config = {
            cloudflare: "{{get_options("admin_captcha_cloudflare_turnstile_website_key","1x00000000000000000000AA")}}",
            recaptcha: "{{get_options("admin_captcha_recaptcha_website_key")}}",
            service: "{{get_options("admin_captcha_service")}}"
        }
        const system_theme = "{{session()->get('theme',session()->get('auto_theme','light'))}}"
        var auto_theme = "{{session()->get('auto_theme','light')}}";
        var csrf_token = "{{ csrf_token() }}";
        var ws_url = "{{ws_url()}}";
        var _token = "{{auth()->token()}}";
        var imageUpUrl = "/user/upload/image?_token={{ csrf_token() }}";
        var fileUpUrl = "/user/upload/file?_token={{ csrf_token() }}";
    </script>
    <meta name="description" content="@yield('description',get_options('description'))">
    <meta name="keywords" content="@yield('keywords',get_options('keywords'))">
    @yield('css')
    @yield('headers')
    {{--插件css--}}
    @foreach((new \App\CodeFec\Plugins())->get_all() as $value)
        @if(file_exists(public_path("plugins/".$value."/".$value.".css")))
            <link href="{{ file_hash("plugins/".$value."/".$value.".css") }}" rel="stylesheet"/>
        @endif
    @endforeach
    @if(get_options('theme_common_diy_code_head'))
        @include(get_component_view_name(get_options('theme_common_diy_code_head')))
    @endif
</head>

<body data-bs-theme="{{session()->get('theme',session()->get('auto_theme','light'))}}">
<div class="page">
    @if(get_options("web_header"))
        @include(get_component_view_name(get_options('web_header')))
    @else
        @include("App::layouts.themes.header-".get_options('core_theme_header',1))
    @endif
    @include("App::layouts.errors")
    @include("App::layouts._msg")
    <div id="{{ path_class() }}-page">
        @yield('header')
        <div class="page-body">
            <div class="container-xl">
                @yield('content')
            </div>
        </div>
    </div>

    @if(!get_options('web_footer'))
        @include("App::layouts.themes.footer-1")
    @else
        @include(get_component_view_name(get_options('web_footer')))
    @endif
    <script src='/js/jquery-3.7.1.min.js'></script>
    <script src="{{ mix('js/vue.js') }}"></script>
    <script src="{{ '/tabler/js/tabler.min.js' }}"></script>
    <script src="{{ mix('plugins/Core/js/app.js') }}"></script>
    {{--    <script src="{{ file_hash('js/diy.js') }}"></script>--}}
    {{-- <!-- 自定义Js --> --}}
    @foreach (\App\CodeFec\Ui\functions::get('js') as $key => $value)
        <script src="{{ $value }}"></script>
    @endforeach
    @yield('scripts')
    {{--插件js--}}
    @foreach((new \App\CodeFec\Plugins())->get_all() as $value)
        @if(file_exists(public_path("plugins/".$value."/".$value.".js")))
            <script src="{{ file_hash("plugins/".$value."/".$value.".js") }}"></script>
        @endif
    @endforeach
    {{--        用户水印--}}
    @if(auth()->check() && get_options('user_username_watermark')==="true" && get_current_action()!==[
"controller" => 'App\Plugins\Core\src\Controller\IndexController',
"method" => 'show'
])
        <script type="text/javascript" src="{{mix('plugins/Topic/js/watermark.js')}}"></script>
        <script>
            $('body').watermark({
                texts: ["{{auth()->data()->username}}","{{date("Y-m-d H:i:s")}}"],
                textColor: "#d2d2d2", //文字颜色
            });
        </script>
    @endif
</div>
@if(get_options('theme_common_diy_code_body'))
    @include(get_component_view_name(get_options('theme_common_diy_code_body')))
@endif
</body>
</html>
