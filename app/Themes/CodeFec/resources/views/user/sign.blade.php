<!DOCTYPE html>
<html lang="zh-CN" data-theme="{{ plugins_core_theme() }}">

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
    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet" />
    <script>
        var csrf_token = "{{ recsrf_token() }}";
        var ws_url = "{{ws_url()}}";
        var login_token = "{{auth()->token()}}";
    </script>
    @yield('css')
    @yield('headers')
</head>

<body class="border-top-wide border-primary d-flex flex-column">
<div class="container-tight py-4">
    @include($view)
</div>

<script src='/js/jquery-3.6.0.min.js'></script>
<script src="{{ mix('js/vue.js') }}"></script>
<script src="{{ '/tabler/js/tabler.min.js' }}"></script>
<script src="{{ mix('plugins/Core/js/app.js') }}"></script>
<script src="{{ file_hash('js/diy.js') }}"></script>
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
<script src="{{ mix('plugins/Core/js/sign.js') }}"></script>
</body>

</html>
