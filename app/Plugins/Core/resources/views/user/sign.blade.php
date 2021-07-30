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
</head>

<body class="antialiased">
@include($view)
<script src="{{ mix('js/vue.js') }}"></script>
<script src="{{ mix('plugins/Core/js/sign.js') }}"></script>
</body>

</html>
