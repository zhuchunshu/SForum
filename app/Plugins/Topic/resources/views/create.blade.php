@extends("plugins.Core.app")

@section('title', '发表帖子')

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            发表帖子
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection
@section('content')

    <div id="create-topic-vue">
        <form action="">
            <div class="row row-cards">
                <div class="col-md-9">
                    <div class="mb-3 border-0 card card-body">
                        <h3 class="card-title">标题</h3>
                        <input type="text" v-model="title" class="form-control form-control-lg form-control-flush" placeholder="请输入标题">
                        <h3 class="card-title">标签</h3>
                        <div class="mb-3">
                            <select id="select-tags" class="form-select form-select-lg form-control-flush">
                                <option v-for="option in tags" :value="option.value">
                                    @{{ option.text }}
                                </option>
                            </select>
                        </div>
                        <div id="content-vditor"></div>
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="row row-cards">
                        @foreach($right as $value)
                            @include($value)
                        @endforeach
                    </div>
                </div>
            </div>
        </form>
    </div>

@endsection

@section('scripts')
    <script>

    </script>
    <script type="text/javascript">
        var imageUpUrl = "/user/upload/image?_token={{csrf_token()}}";
    </script>
    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
