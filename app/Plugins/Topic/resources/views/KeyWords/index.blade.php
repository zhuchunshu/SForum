@extends("App::app")

@section('title', '关键词列表')
@section('description', '本站帖子关键词列表')
@section('keywords', '本站帖子关键词列表')

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
                关键词列表
            </h2>
        </div>


    </div>
</div>
</div>
@endsection

@section('content')

<div class="row row-cards">
@if ($page->count())
    @foreach ($page as $value)
        <div class="col-4 col-md-2">
            <div class="keywords-a" style="background-color:{{$color[array_rand($color)]}}">
                <a href="/keywords/{{$value->name}}.html">
                    <h2 title="{{$value->name}}">{{$value->name}}</h2>
                    <p>共{{count($value->kw)}}篇文章</p>
                </a>
            </div>
        </div>
    @endforeach
@else
<div class="col-md-4">
    <a class="border-0 card">
        <div class="card-status-top bg-danger"></div>
        <div class="card-body">
            <div class="row">
                <div class="col-auto">
                    <span class="avatar"></span>
                </div>
                <div class="col">
                    <h3 class="card-title text-h2">暂无内容</h3>
                </div>
            </div>
        </div>
    </a>
</div>
@endif
{!! make_page($page) !!}
</div>
</div>

@endsection

@section('headers')
        <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection