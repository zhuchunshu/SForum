@extends('App::app')
@section('title','「'.$data->title.'」的文档内容')
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
                            {{$data->docsClass->name}}
                        </h2>
                    </div>

                    @if(auth()->id()===(int)$data->user_id && Authority()->check("docs_create"))
                        <div class="col-auto mb-1">
                            <a href="/docs/create/{{$data->docsClass->id}}" class="btn btn-dark">发布文档</a>
                        </div>
                    @endif

                    <div class="col-auto mb-1">
                        @if(auth()->id()===(int)$data->user_id && Authority()->check("docs_edit"))
                            <a href="/docs/editClass/{{$data->id}}" class="btn btn-dark">修改分类</a>
                        @elseif(Authority()->check("admin_docs_edit"))
                            <a href="/docs/editClass/{{$data->id}}" class="btn btn-dark">修改分类</a>
                        @endif
                    </div>
                    <div class="col-auto mb-1" id="vue-docs-class-show-footer">

                        @if(auth()->id()===(int)$data->user_id && Authority()->check("docs_delete"))
                            <button class="btn btn-dark" @@click="docs_delete_class">删除分类</button>
                        @elseif(Authority()->check("admin_docs_delete"))
                            <button class="btn btn-dark" @@click="docs_delete_class">删除分类</button>
                        @endif
                    </div>

                    <div class="col-auto mb-1">
                        @if((int)$data->user_id===auth()->id() && Authority()->check("docs_edit"))
                            <a href="/docs/edit/{{$data->id}}" class="btn btn-dark">修改文档</a>
                        @elseif(Authority()->check("admin_docs_edit"))
                            <a href="/docs/edit/{{$data->id}}" class="btn btn-dark">修改文档</a>
                        @endif

                    </div>
                    <div class="col-auto mb-1" id="docs-app">
                        @if(auth()->check())
                            @if((int)$data->user_id===auth()->id() && Authority()->check("docs_delete"))
                                <button class="btn btn-dark" @@click="docs_delete({{$data->id}})">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-trash" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                        <line x1="4" y1="7" x2="20" y2="7"></line>
                                        <line x1="10" y1="11" x2="10" y2="17"></line>
                                        <line x1="14" y1="11" x2="14" y2="17"></line>
                                        <path d="M5 7l1 12a2 2 0 0 0 2 2h8a2 2 0 0 0 2 -2l1 -12"></path>
                                        <path d="M9 7v-3a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v3"></path>
                                    </svg>
                                    删除文档
                                </button>
                            @elseif(Authority()->check("admin_docs_delete"))
                                <button class="btn btn-dark" @@click="docs_delete({{$data->id}})">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-trash" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                        <line x1="4" y1="7" x2="20" y2="7"></line>
                                        <line x1="10" y1="11" x2="10" y2="17"></line>
                                        <line x1="14" y1="11" x2="14" y2="17"></line>
                                        <path d="M5 7l1 12a2 2 0 0 0 2 2h8a2 2 0 0 0 2 -2l1 -12"></path>
                                        <path d="M9 7v-3a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v3"></path>
                                    </svg>
                                    删除文档
                                </button>
                            @endif
                        @endif
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection


@section('scripts')
    <script>var docs_class_id = {{$data->docsClass->id}};</script>
    <script src="{{file_hash("plugins/Docs/js/docs.js")}}"></script>
    <script>
        $(function () {
            $('a[docs-menu="active"]').each(function () {
                $(this).parents('ul').addClass('show');
            });
        })
    </script>
@endsection

@section('content')

    <div class="row gx-lg-5">
        @include('Docs::menu')
        <div class="col-lg-9">
            <div class="card card-lg">
                <div class="mt-3 mx-3">
                    <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                        <li class="breadcrumb-item"><a href="/docs">文档</a></li>
                        <li class="breadcrumb-item"><a href="/docs/{{$data->class_id}}">{{$data->docsClass->name}}</a>
                        </li>
                        <li class="breadcrumb-item active" aria-current="page"><a href="#">{{$data->title}}</a></li>
                    </ol>
                </div>
                <div class="card-body">
                    <div class="markdown">
                        {!! $data->content !!}
                    </div>
                </div>
            </div>
            <div class="mt-3">
                <div class="border-0 card">
                    <div class="card-body">
                        <ul class="pagination ">

                            @if ($shang)
                                <li class="page-item page-prev">
                                    <a class="page-link"
                                       href="/docs/{{$data->class_id}}/{{$shang['id']}}.html">
                                        <div class="page-item-subtitle">上一篇文档</div>
                                        <div class="page-item-title">
                                            {{ \Hyperf\Utils\Str::limit($shang['title'], 20, '...') }}</div>
                                    </a>
                                </li>
                            @else
                                <li class="page-item page-prev disabled">
                                    <a class="page-link" href="#" tabindex="-1" aria-disabled="true">
                                        <div class="page-item-subtitle">上一篇文档</div>
                                        <div class="page-item-title">暂无</div>
                                    </a>
                                </li>
                            @endif
                            @if ($xia)
                                <li class="page-item page-next">
                                    <a class="page-link"
                                       href="/docs/{{$data->class_id}}/{{ $xia['id']  }}.html">
                                        <div class="page-item-subtitle">下一篇文档</div>
                                        <div class="page-item-title">
                                            {{ \Hyperf\Utils\Str::limit($xia['title'], 20, '...') }}</div>
                                    </a>
                                </li>
                            @else
                                <li class="page-item page-next disabled">
                                    <a class="page-link" href="#">
                                        <div class="page-item-subtitle">下一篇文档</div>
                                        <div class="page-item-title">暂无</div>
                                    </a>
                                </li>
                            @endif
                        </ul>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection
