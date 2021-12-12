@extends("Core::app")
@section('title','本站举报')
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
                            举报列表
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection
@section('content')
    <div class="row justify-content-center">
        <div class="col-md-8">
            @if($page->count())
                <div class="row row-cards">
                    @foreach($page as $value)
                        <div class="col-md-12">
                            <div class="card border-0">
                                <div class="card-body">
                                    <div class="row">
                                        <div class="col-md-12">
                                            <div class="row">
                                                <div class="col-auto">
                                                    <span style="background-image: url({{super_avatar($value->user)}})" class="avatar"></span>
                                                </div>
                                                <div class="col">
                                                    <div class="col">
                                                        <div class="topic-author-name">
                                                            <a href="/users/{{$value->user->username}}.html" class="text-reset">{{$value->user->username}}</a>
                                                        </div>
                                                        <div>举报创建于:{{format_date($value->created_at)}}</div>
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="col-md-12">
                                            <div class="hr-text">举报标题</div>
                                            <a href="/report/{{$value->id}}.html" class="card-title">{{$value->title}}</a>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    @endforeach

                </div>
            @else
                <div class="row">
                    <div class="col-md-12">
                        <div class="card border-0">
                            <div class="card-body">
                                <h3 class="card-title">暂无内容</h3>
                            </div>
                        </div>
                    </div>

                </div>
            @endif
            {!! make_page($page) !!}
        </div>
    </div>
@endsection