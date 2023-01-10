@extends('App::app')
@section('title','在「'.$data->name.'」下发布文档')
@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12" id="docs-create">
            <div class="border-0 card card-body">
                <h3 class="card-title">发布文档</h3>
                <form method="post" enctype="multipart/form-data" @@submit.prevent="submit">
                    <x-csrf/>
                    <div class="mb-3">
                        <div class="row">
                            <div class="col col-auto">
                                <span class="avatar" style="background-image: url('{{$data->icon}}')"></span>
                            </div>
                            <div class="col">
                                <h3 class="text-reset text-black">{{$data->name}}</h3>
                                <span>创建于:{{format_date($data->created_at)}}</span>
                            </div>
                        </div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">标题</label>
                        <input type="text" class="form-control" v-model="title">
                    </div>
                    <div class="mb-3">
                        <div id="docs-editor"></div>
                    </div>
                    <div class="mb-3">
                        <button class="btn btn-primary" type="submit">提交</button>
                    </div>
                </form>
            </div>
        </div>
    </div>



@endsection

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
                            发布文档
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script>var docs_class_id = {{$data->id}}; var imageUpUrl = "/user/upload/image?_token={{ csrf_token() }}";</script>
    <script src="{{file_hash("plugins/Docs/js/docs.js")}}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection