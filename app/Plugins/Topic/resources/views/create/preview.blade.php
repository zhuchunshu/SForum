<!doctype html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport"
          content="width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0">
    <meta http-equiv="X-UA-Compatible" content="ie=edge">

    <title>预览</title>
    <link rel="icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link rel="shortcut icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet"/>
    <link href="{{ '/tabler/css/tabler-vendors.min.css' }}" rel="stylesheet"/>
    <link rel="stylesheet" href="{{mix("plugins/Core/css/core.css")}}">
    {{--    <link href="{{ file_hash("css/diy.css") }}" rel="stylesheet"/>--}}
    <link rel="stylesheet" href="{{mix('css/app.css')}}">
    <script>

        var auto_theme = "{{session()->get('auto_theme','light')}}";
        var csrf_token = "{{ csrf_token() }}";
        var ws_url = "{{ws_url()}}";
        var _token = "{{auth()->token()}}";
        var imageUpUrl = "/user/upload/image?_token={{ csrf_token() }}";
        var fileUpUrl = "/user/upload/file?_token={{ csrf_token() }}";
    </script>
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
    <link rel="stylesheet" href="{{file_hash('tabler/libs/plyr/dist/plyr.css')}}">
    <link rel="stylesheet" href="{{ file_hash('css/prism.css') }}">
</head>
<body class="border-0 card card-body {{session()->get('theme','antialiased')}}" style="background-color: transparent;">
<article class="col-md-12 article markdown" id="topic-content">
    {!! ContentParse()->parse($content) !!}
</article>

<script src='/js/jquery-3.6.0.min.js'></script>
<script src="{{ mix('js/vue.js') }}"></script>
<script src="{{ '/tabler/libs/apexcharts/dist/apexcharts.min.js' }}"></script>
<script src="{{ '/tabler/js/tabler.min.js' }}"></script>
{{--<script src="{{ file_hash('js/diy.js') }}"></script>--}}
<script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
<script src="{{mix('plugins/Topic/js/core.js')}}"></script>
<script src="{{ file_hash('js/prism.js') }}"></script>
<script>
    document.addEventListener("DOMContentLoaded", function () {
        window.Plyr && (new Plyr('video'));
    });
</script>
</body>
</html>