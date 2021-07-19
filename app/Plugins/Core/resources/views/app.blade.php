<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{ plugins_core_theme() }}">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
        content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>@yield("title","标题") - {{ get_options('title', config('app_name', 'CodeFec')) }}</title>
    <link rel="stylesheet" href="{{ mix('plugins/Core/css/app.css') }}">
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <script>
        var csrf_token = "{{ csrf_token() }}";
    </script>
    <meta name="description" content="{{ get_options('description') }}">
    <meta name="keywords" content="{{ get_options('keywords') }}">
    <!-- 自定义CSS -->
    @foreach (\App\CodeFec\Ui\functions::get('css') as $key => $value)
        <link rel="stylesheet" href="{{ $value }}">
    @endforeach
    @yield('css')
</head>

<body class="antialiased">
    <div class="flex flex-col h-screen justify-between">
        @include("plugins.Core.layouts.themes.header-".get_options('core_theme_header',1))
        <main class="mb-auto h-10">
            <div id="app" class="container mx-auto {{ path_class() }}-page lg:pt-2">
                <div class="flex">
                    <div class="lg:w-1/6 lg:block lg:pr-4">
                        @include('plugins.Core.layouts.themes.menu')
                    </div>
                    <div class="px-2 w-full lg:px-0 lg:w-5/6 lg:pl-4">
                        @yield('content')
                    </div>
                </div>
            </div>
        </main>
        @include("plugins.Core.layouts.themes.footer-".get_options('core_theme_footer',1))
    </div>
    <script defer src="{{ mix('js/vue.js') }}"></script>
    <script defer src="{{ mix('js/alpine.js') }}"></script>
    @if (get_options('theme_common_require_mithril', 'yes') != 'no')
        <script src="{{ mix('plugins/Core/js/mithril.js') }}"></script>
    @endif
    {{-- <!-- 自定义Js --> --}}
    @foreach (\App\CodeFec\Ui\functions::get('js') as $key => $value)
        <script src="{{ $value }}"></script>
    @endforeach
    @yield('scripts')
</body>

</html>
