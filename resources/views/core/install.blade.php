<!Doctype html>
<html lang="zh-CN">

<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover" />
    <meta http-equiv="X-UA-Compatible" content="ie=edge" />
    <title>SForum安装 - {{ config('codefec.app.name') }}</title>
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

<body class="antialiased border-top-wide border-primary d-flex flex-column" id="app-install">
    <div class="page page-center">
        <div class="container-tight py-4">
            <div class="text-center mb-4">
                <a href="#"><img src="/logo.svg" height="36" alt=""></a>
            </div>
            <div class="card card-md">
                <div class="card-body text-center">
                    <h1>安装SForum!</h1>
                    <p class="text-muted">本项目开源地址: <a href="https://github.com/zhuchunshu">https://github.com/zhuchunshu</a><br>安装过程中如若遇到问题请到论坛反馈:
                        <a href="https://www.runpod.cn">https://www.runpod.cn</a></p>
                </div>
                <div class="hr-text hr-text-center hr-text-spaceless">@{{ tips }}</div>
                <div class="card-body">
                    @include('core.install.tips')
                    @include('core.install.user')

                </div>
            </div>
            <div class="row align-items-center mt-3">
                <div class="col-4">
                    <div class="progress">
                        <div class="progress-bar" :style="{width: progress + '%' }" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
                            <span class="visually-hidden">@{{ progress }}% Complete</span>
                        </div>
                    </div>
                </div>
                <div class="col">
                    <div class="btn-list justify-content-end" v-if="install_lock">
                        <a  v-if="install_lock<5" href="#" @@click="next" class="btn btn-primary">
                            下一步
                        </a>
                        <a  v-if="install_lock>=5" href="#" data-bs-toggle="modal" data-bs-target="#modal-scrollable"  class="btn btn-primary">
                            完成安装
                        </a>
                    </div>
                </div>
            </div>
        </div>
    </div>
    <div class="modal modal-blur fade" id="modal-scrollable" tabindex="-1" role="dialog" aria-hidden="true">
        <div class="modal-dialog  modal-xl modal-dialog-centered modal-dialog-scrollable" role="document">
            <div class="modal-content">
                <div class="modal-header">
                    <h5 class="modal-title">SForum免责声明</h5>
                    <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                </div>
                <div class="modal-body">
                    @include("core.install.disclaimer")
                </div>
                <div class="modal-footer">
                    <button type="button" class="btn me-auto" data-bs-dismiss="modal">Close</button>
                    <button type="button" class="btn btn-primary" @@click="install" data-bs-dismiss="modal">签署并安装</button>
                </div>
            </div>
        </div>
    </div>
    <script src="{{ mix('js/vue.js') }}"></script>
    <script src="{{ mix('js/install.js') }}"></script>
    <script src="{{ '/tabler/libs/apexcharts/dist/apexcharts.min.js' }}"></script>
    <script src="{{ '/tabler/js/tabler.min.js' }}"></script>
</body>

</html>
