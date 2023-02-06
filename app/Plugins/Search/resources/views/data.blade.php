@extends("App::app")

@section('title',__("app.search result",['search'=>"「".$q."]"]))
@section('description',__("app.search result",['search'=>"「".$q."]"]))

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
                            {{ __("app.search result",['search'=>"「".$q."]"]) }}
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-lg-9">
                    @if($page->count())
                        <div class="row row-cards">
                            @foreach($page as $data)
                                <article class="col-md-12">
                                    <div class="card">
                                        <div class="card-header">
                                            <a href="{{$data['url']}}" class="card-title link-primary">
                                                {{$data['title']}}
                                            </a>
                                        </div>
                                        <div class="card-body markdown">
                                            {!! $data['content'] !!}
                                        </div>
                                        <div class="card-footer">
                                            <ul style="margin-left: -20px">
                                                <li style="float: left;list-style: outside none none;padding: 3px;line-height: 1.6">
                                                    <a href="{{$data['user']['url']}}">{{$data['user']['name']}}</a>
                                                </li>
                                                <li style="float: left;list-style: outside none none;padding: 3px;line-height: 1.6">
                                                    <a href="{{$data['tag']['url']}}">{{$data['tag']['name']}}</a>
                                                </li>
                                                <li style="float: left;list-style: outside none none;padding: 3px;line-height: 1.6">
                                                    <span class="text-muted">{{format_date($data['created_at'])}}</span>
                                                </li>
                                            </ul>
                                        </div>
                                    </div>
                                </article>
                            @endforeach
                        </div>
                        <div class="mt-3">
                            @if($page->hasPages())
                                {!! make_page($page) !!}
                            @endif
                        </div>
                    @else
                        <div class="card card-body empty">
                            <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0" /><path d="M9 10l.01 0" /><path d="M15 10l.01 0" /><path d="M9.5 15.25a3.5 3.5 0 0 1 5 0" /></svg>
                            </div>
                            <p class="empty-title">No results found</p>
                            <p class="empty-subtitle text-muted">
                                无所搜结果,建议换一个词重试
                            </p>
{{--                            <div class="empty-action">--}}
{{--                                <a href="#" class="btn btn-primary">--}}
{{--                                    <!-- Download SVG icon from http://tabler-icons.io/i/search -->--}}
{{--                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 10m-7 0a7 7 0 1 0 14 0a7 7 0 1 0 -14 0" /><path d="M21 21l-6 -6" /></svg>--}}
{{--                                    Search again--}}
{{--                                </a>--}}
{{--                            </div>--}}
                        </div>
                    @endif
                </div>
                <div class="col-lg-3">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('App::index.right')
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
    <script src="/tabler/libs/apexcharts/dist/apexcharts.min.js"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
