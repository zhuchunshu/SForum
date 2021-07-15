<!DOCTYPE html>
<html lang="zh-CN" data-theme="@yield('data-theme','light')">

<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">
    <title>@yield("title","标题") - {{ config('app_name', 'CodeFec') }}</title>
    <link rel="stylesheet" href="{{ mix('plugins/Core/css/app.css') }}">
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <script>var csrf_token="{{csrf_token()}}";</script>
    <!-- 自定义CSS -->
    @foreach(\App\CodeFec\Ui\functions::get("css") as $key => $value)
        <link rel="stylesheet" href="{{ $value }}">
    @endforeach
    @yield('css')
</head>

<body class="antialiased">
<div id="app" class="wrapper {{ path_class() }}-page">
    {{-- @include('layouts.bujian._msg')
    @include('shared._error') --}}
    @yield('content')
</div>

<script src="{{ mix('js/vue.js') }}"></script>
<!-- 自定义Js -->
@foreach(\App\CodeFec\Ui\functions::get("js") as $key => $value)
    <script src="{{ $value }}"></script>
@endforeach
@yield('scripts')
</body>

</html>
