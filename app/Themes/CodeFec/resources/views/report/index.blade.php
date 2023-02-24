@extends("App::app")
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
        <div class="col-md-12">
            @if($page->count())
                <div class="row row-cards">
                    @foreach($page as $value)
                        <div class="col-md-12">
                            <div class="card">
                                @if($value->status==="pending")
                                    <div class="ribbon bg-indigo">待办</div>
                                @elseif($value->status==="reject")
                                    <div class="ribbon bg-red">驳回</div>
                                @elseif($value->status==="approve")
                                    <div class="ribbon bg-green">批准</div>
                                @endif
                                <div class="card-header">
                                    <div class="row">
                                        <div class="col-auto">
                                            <span style="background-image: url({{super_avatar($value->user)}})" class="avatar"></span>
                                        </div>
                                        <div class="col">
                                            <div class="col">
                                                <div class="topic-author-name">
                                                    <a href="/users/{{$value->user->id}}.html" class="text-reset">{{$value->user->username}}</a>
                                                </div>
                                                <div>{{__('app.report created at',['time' => format_date($value->created_at)])}}</div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                                <div class="card-body">
                                    <a href="/report/{{$value->id}}.html" class="card-title">{{$value->title}}</a>
                                </div>
                                <div class="card-footer">
                                    <div class="d-flex flex-row-reverse">
                                        <a href="/report/{{$value->id}}.html" class="btn btn-primary">查看</a>
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
                                <h3 class="card-title">{{__("app.No more results")}}</h3>
                            </div>
                        </div>
                    </div>

                </div>
            @endif
            {!! make_page($page) !!}
        </div>
    </div>
@endsection