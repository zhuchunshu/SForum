<!doctype html>
<html lang="zh-CN">

<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>Error - {{ get_options('title', config('app_name', 'CodeFec'))}}</title>
    <link rel="preconnect" href="https://www.google-analytics.com" crossorigin>
    <link rel="preconnect" href="https://fonts.googleapis.com" crossorigin>
    <link rel="preconnect" href="https://www.googletagmanager.com" crossorigin>
    <meta name="msapplication-TileColor" content="#206bc4"/>
    <meta name="theme-color" content="#206bc4"/>
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent"/>
    <meta name="apple-mobile-web-app-capable" content="yes"/>
    <meta name="mobile-web-app-capable" content="yes"/>
    <meta name="HandheldFriendly" content="True"/>
    <meta name="MobileOptimized" content="320"/>
    <link rel="icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <link rel="shortcut icon" href="{{get_options('theme_common_icon','/logo.svg')}}" type="image/x-icon"/>
    <!-- CSS files -->
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet"/>
    <script>
        var theme_status = @if(session()->has('theme')) {{"true"}} @else {{"false"}} @endif;
        const captcha_config = {
            cloudflare: "{{get_options("admin_captcha_cloudflare_turnstile_website_key","1x00000000000000000000AA")}}",
            recaptcha: "{{get_options("admin_captcha_recaptcha_website_key")}}",
            service: "{{get_options("admin_captcha_service")}}"
        }
        const system_theme = "{{session()->get('theme',session()->get('auto_theme','light'))}}"
        var auto_theme = "{{session()->get('auto_theme','light')}}";
    </script>

</head>

<body data-bs-theme="{{session()->get('theme',session()->get('auto_theme','light'))}}"
      class="antialiased border-top-wide border-primary d-flex flex-column">
<div class="page page-center">
    <div class="container-tight py-4">
        <div class="empty" id="empty">
            <div class="empty-header">{{ $code }}</div>
            <p class="empty-title">{{ $data['msg'] }}</p>
            <p class="empty-subtitle text-muted">
                Sorry, you are temporarily unable to access this page
            </p>
            <div class="empty-action">
                <a href="{{$redirect}}?redirect={{request()->getUri()}}" class="btn btn-primary">
                    <!-- Download SVG icon from http://tabler-icons.io/i/arrow-left -->
                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24"
                         stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                         stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                        <line x1="5" y1="12" x2="19" y2="12"/>
                        <line x1="5" y1="12" x2="11" y2="18"/>
                        <line x1="5" y1="12" x2="11" y2="6"/>
                    </svg>
                    Return
                </a>
            </div>
        </div>
    </div>
</div>
<!-- Tabler Core -->
<script src="{{ '/tabler/js/tabler.min.js' }}"></script>
</body>

</html>
