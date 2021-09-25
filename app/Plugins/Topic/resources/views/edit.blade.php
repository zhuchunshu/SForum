@extends("plugins.Core.app")

@section('title', '修改帖子:'.$data['title'])

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
                            修改帖子
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection
@section('content')

    <div id="edit-topic-vue">
        <form action="" method="post" @@submit.prevent="submit">
            <div class="row row-cards">
                <div class="col-md-9">
                    <div class="mb-3 border-0 card card-body">
                        <h3 class="card-title">标题</h3>
                        <input type="text" v-model="title" class="form-control form-control-lg form-control-flush"
                               placeholder="请输入标题" required>
                        <h3 class="card-title">标签</h3>
                        <div class="mb-3">
                            <select id="select-tags" v-model="tag_selected"
                                    class="form-select form-select-lg form-control-flush">
                                <option v-for="option in tags" :data-custom-properties="option.icons" :value="option.value">
                                    @{{ option . text }}
                                </option>
                            </select>
                        </div>
                        <div class="vditor-superf-toolbar">
                            <div class="vditor-toolbar vditor-toolbar--pin" style="padding-left: 10px;">
                                <div class="vditor-toolbar__item">
                                    @include('plugins.Topic.create-toolbar')
                                </div>
                                {{--                                    <span class="vditor-counter vditor-tooltipped vditor-tooltipped__nw"--}}
                                {{--                                    aria-label="已写字数">18</span>--}}
                            </div>
                        </div>
                        <div class="mb-3">
                            <div id="content-vditor"></div>
                        </div>
                        <div class="mb-3">
                            <button class="btn btn-primary">发布</button> Or <button type="button" @@click="draft" class="btn btn-danger">存为草稿</button>
                        </div>
                    </div>
                </div>
                <div class="col-md-3">
                    <div class="row row-cards">
                        @foreach ($right as $value)
                            @include($value)
                        @endforeach
                    </div>
                </div>
            </div>
        </form>
    </div>

@endsection

@section('scripts')
    <script type="text/javascript">
        var imageUpUrl = "/user/upload/image?_token={{ csrf_token() }}";
        var topic_id = {{$data->id}};
    </script>

    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
