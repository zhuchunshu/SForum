@extends('Core::app')

@section('title',$user->username.' 的博客')

@section('content')

    <div class="col-md-12">

        <div class="row row-cards">

            <div class="col-md-9">
                @if($page->count())
                    <div class="row row-cards">
                        @foreach($page as $data)
                            <posts class="col-md-12">
                                <div class="border-0 card card-body">
                                    <div class="row">
                                        <div class="col-md-12">
                                            <div class="row">
                                                <div class="col-auto">
                                                    <a href="/users/{{$user->username}}.html" class="avatar" style="background-image: url({{super_avatar($user)}})">

                                                    </a>
                                                </div>
                                                <div class="col">
                                                    <a href="/users/{{$user->username}}.html" style="margin-bottom:0;text-decoration:none;" class="card-title text-reset">{{$user->username}}</a>
                                                    <div style="margin-top:1px">发布于:{{format_date($data->created_at)}}</div>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="col-md-12">
                                            <div class="row">
                                                <div class="col-md-12 markdown home-article">
                                                    <a href="/blog/article/{{$data->id}}.html" class="text-reset">
                                                        <h2>{{$data->title}}</h2>
                                                    </a>
                                                    <span class="home-summary">{{\Hyperf\Utils\Str::limit(strip_tags($data->content),300)}}</span>
                                                </div>
                                            </div>
                                        </div>

                                        <div class="col-md-12 text-center">
                                            <a href="/blog/article/{{$data->id}}.html"><b class="text-primary">查看全文</b></a>
                                        </div>
                                    </div>
                                </div>
                            </posts>
                        @endforeach
                    </div>
                @else
                    暂无内容
                @endif
                <div style="margin-top:10px">
                    {!! make_page($page) !!}
                </div>
            </div>


            <div class="col-md-3">
                <div class="card">
                    <div class="card-status-top bg-primary"></div>
                    <div class="card-body">
                        <h3 class="card-title">
                            {{$user->username}} 博客
                        </h3>
                    </div>
                    <div class="card-footer">
                        <a href="/blog/article/create" class="btn btn-dark">写文章</a>
                        <a href="/blog/class" class="btn btn-light">分类管理</a>
                    </div>
                </div>
            </div>
        </div>

    </div>

@endsection