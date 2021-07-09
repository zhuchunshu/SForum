<!doctype html>
<html lang="zh-CN">

<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>@yield("title","第一步") - 安装 - {{ config('codefec.app.name') }}</title>
    <meta name="theme-color" content="#206bc4" />
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent" />
    <meta name="apple-mobile-web-app-capable" content="yes" />
    <meta name="mobile-web-app-capable" content="yes" />
    <meta name="HandheldFriendly" content="True" />
    <meta name="MobileOptimized" content="320" />
    <link rel="icon" href="/logo.svg" type="image/x-icon" />
    <link rel="shortcut icon" href="/logo.svg" type="image/x-icon" />
    <!-- CSS files -->
    <link href="{{ '/tabler/css/tabler.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-flags.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-payments.min.css' }}" rel="stylesheet" />
    <link href="{{ '/tabler/css/tabler-vendors.min.css' }}" rel="stylesheet" />
    <script>
        var csrf_token = "{{ csrf_token() }}";
    </script>
</head>

<body class="antialiased border-top-wide border-primary d-flex flex-column">
    <div class="page page-center">
        <div class="container-tight py-4">
            <div class="text-center mb-4">
                <a href="#"><img src="/logo.svg" height="36" alt=""></a>
            </div>
            <form action="@yield('action','/install?step=1')" method="POST">
                <x-csrf/>
                <div class="card card-md">
                    <div class="card-body text-center py-4 p-sm-5">
                        <img src="@yield('img','/install/step1.svg')" height="128" class="mb-n2" height="120" alt="">
                        <h1 class="mt-5">Welcome to SuperForum!</h1>
                        <p class="text-muted">仅需简单几步,完成程序安装</p>
                    </div>
                    <div class="hr-text hr-text-center hr-text-spaceless">@yield('hr','your data')</div>
                    <div class="card-body">

                        @yield('content')
                    </div>
                </div>
                <div class="row align-items-center mt-3">
                    @yield('footer')
                </div>
            </form>
        </div>
    </div>

    <script src="{{ mix('js/vue.js') }}"></script>
    <script src="{{ mix('js/app.js') }}"></script>
    <script src="{{ '/tabler/libs/apexcharts/dist/apexcharts.min.js' }}"></script>
    <!-- Tabler Core -->
    <script src="{{ '/tabler/js/tabler.min.js' }}"></script>
</body>

</html>
