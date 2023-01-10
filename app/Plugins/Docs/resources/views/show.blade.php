@extends('App::app')
@section('title','「'.$data->name.'」的文档信息')
@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Documentation
                        </div>
                        <h2 class="page-title">
                            {{$data->name}}
                        </h2>
                    </div>

                    <div class="col-auto">
                        <a href="/docs/create/{{$data->id}}" class="btn btn-dark">发布文档</a>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection


@section('scripts')
    <script>var docs_class_id = {{$data->id}};</script>
    <script src="{{file_hash("plugins/Docs/js/docs.js")}}"></script>
@endsection

@section('content')

    <div class="row gx-lg-5">
        @include('Docs::menu')
        <div class="col-lg-9">
            <div class="card card-lg">
                <div class="card-body">
                    <div class="markdown">
                        <div>
                            <div class="d-flex mb-3">
                                <h1 class="m-0">{{$data->name}}</h1>
                            </div>
                        </div>
                        <p>这里暂时没有文档,或许... 懒已经成为了一种习惯。</p>

                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection


