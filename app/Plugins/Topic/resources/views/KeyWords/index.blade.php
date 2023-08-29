@extends("App::app")

@section('title', '标签列表')
@section('description', '本站帖子标签列表')
@section('keywords', '本站帖子标签列表')

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
                标签列表
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
<div class="col-md-12">
    <div class="empty">
        <div class="empty-icon">
            <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                <circle cx="12" cy="12" r="9" />
                <line x1="9" y1="10" x2="9.01" y2="10" />
                <line x1="15" y1="10" x2="15.01" y2="10" />
                <path d="M9.5 15.25a3.5 3.5 0 0 1 5 0" />
            </svg>
        </div>
        <p class="empty-title">No results found</p>
        <p class="empty-subtitle text-muted">
            Try adjusting your search or filter to find what you're looking for.
        </p>
    </div>
</div>
@endif
{!! make_page($page) !!}
</div>
</div>

@endsection

@section('headers')
        <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection