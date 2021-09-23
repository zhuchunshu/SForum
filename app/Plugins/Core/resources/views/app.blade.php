<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{ plugins_core_theme() }}">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
        content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>
        @if(request()->path()==="/")
            @if(get_options('home_title'))
                {{get_options('home_title')}}
            @else
                @yield("title","标题") - {{ get_options('title', config('app_name', 'CodeFec')) }}
            @endif
        @else
            @yield("title","标题") - {{ get_options('title', config('app_name', 'CodeFec')) }}
        @endif
    </title>
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-flags.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-payments.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-vendors.min.css' }}" rel="stylesheet" />
    <link rel="stylesheet" href="{{mix("plugins/Core/css/core.css")}}">
    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet" />
    <script>
        var csrf_token = "{{ recsrf_token() }}";
    </script>
    <meta name="description" content="@yield('description',get_options('description'))">
    <meta name="keywords" content="@yield('keywords',get_options('keywords'))">
    @yield('css')
    @yield('headers')
</head>

<body class="antialiased">
@include("plugins.Core.layouts.themes.header-".get_options('core_theme_header',1))
@include("plugins.Core.layouts.errors")
@include("plugins.Core.layouts._msg")
@yield('header')
<div class="page-body">
    <div class="container-xl">
        @yield('content')
    </div>
        @include("plugins.Core.layouts.themes.footer-".get_options('core_theme_footer',1))
    </div>
    <script src='/js/jquery-3.6.0.min.js'></script>
    <script src="{{ mix('js/vue.js') }}"></script>
    <script src="{{ mix('js/alpine.js') }}"></script>
    <script src="{{ '/tabler/libs/apexcharts/dist/apexcharts.min.js' }}"></script>
    <script src="{{ '/tabler/js/tabler.min.js' }}"></script>
    @if (get_options('theme_common_require_mithril', 'yes') !== 'no')
        <script src="{{ mix('plugins/Core/js/mithril.js') }}"></script>
    @endif
    <script src="{{ mix('plugins/Core/js/app.js') }}"></script>
    <script src="{{ file_hash('js/diy.js') }}"></script>
    {{-- <!-- 自定义Js --> --}}
    @foreach (\App\CodeFec\Ui\functions::get('js') as $key => $value)
        <script src="{{ $value }}"></script>
    @endforeach
    @yield('scripts')
</body>

</html>
