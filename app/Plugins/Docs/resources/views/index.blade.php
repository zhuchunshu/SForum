@extends('App::app')
@section('title','本站文档')

@section('content')

            <div class="row gx-lg-5">
                @include('Docs::menu')
                <div class="col-lg-9">
                    <div class="card card-lg">
                        <div class="card-body">
                            <div class="markdown">
                                <div>
                                    <div class="d-flex mb-3">
                                        <h1 class="m-0">Documentation</h1>
                                    </div>
                                </div>
                                <p>关于什么是文档，我也不想做太多介绍。总之，读了它，可以让你学到很多东西！</p>
                                <div class="mt-4">
                                    <div class="row">
                                        @foreach($docs as $id => $data)
                                            <div class="col-sm-6">
                                                <h3><a class="text-linkedin" href="/docs/{{$id}}">#</a> {{$data['name']}}</h3>
                                                <ul class="list-unstyled">
                                                    @if(count($data['docs']))
                                                        @foreach($data['docs'] as $doc)
                                                            <li>
                                                                - <a href="/docs/{{$id}}/{{$doc['id']}}.html">{{$doc['title']}}</a>
                                                            </li>
                                                        @endforeach
                                                            <li>
                                                                - <a href="/docs/{{$id}}">查看全部</a>
                                                            </li>
                                                    @else
                                                        <li>
                                                           - 暂无文档
                                                        </li>
                                                    @endif
                                                </ul>
                                            </div>
                                        @endforeach
                                    </div>
                                </div>
                            </div>
                        </div>
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
                            本站文档
                        </h2>
                    </div>

                    @if(auth()->check() && Authority()->check('docs_create'))
                        <div class="col-auto">
                            <a href="/docs/create.class" class="btn btn-dark">创建文档</a>
                        </div>
                    @endif


                </div>
            </div>
        </div>
    </div>
@endsection
