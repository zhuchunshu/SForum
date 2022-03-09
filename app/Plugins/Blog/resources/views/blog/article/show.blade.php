@extends('Core::app')

@section('title',$data->title.' - '.$user->username.'的博客')

@section('content')

    <div class="col-md-12">

        <div class="row row-cards justify-content-center">

            <div class="col-md-9">

                <article class="border-0 card">
                    <div class="card-header">
                        <h1 style="margin-bottom:-3px">
                            {{$data->title}}
                        </h1>
                    </div>
                    <div class="card-body vditor-reset">
                        {!! $data->content !!}
                    </div>
                </article>

            </div>

            <div class="col-md-3">

                <div class="row row-cards">

                    <div class="col-md-12">
                        <div class="border-0 card">
                            <div class="card-header">
                                <h3 class="card-title"><a href="/blog/{{$user->username}}.html">{{$user->username}} 的博客</a></h3>
                            </div>
                            <div class="card-body">
                                {{$user->username}}博客 创建于:{{format_date($blog->created_at)}}
                            </div>
                            @if($quanxian===true)
                                <div class="card-footer">
                                    <a class="btn btn-primary" onclick="if(confirm('确定要删除吗? 删除后不可恢复')){location.href='/blog/article/{{$data->id}}/remove?_token={{csrf_token()}}';}">删除</a>
                                    <a href="/blog/article/{{$data->id}}/edit" class="btn btn-light">修改</a>
                                </div>
                            @endif
                        </div>
                    </div>

                </div>

            </div>

        </div>

    </div>

@endsection

@section('scripts')
    <script src="{{file_hash('plugins/Blog/js/article.js')}}"></script>
@endsection
@section('headers')
    <link rel="stylesheet" href="{{file_hash('plugins/Blog/css/article.css')}}">
@endsection