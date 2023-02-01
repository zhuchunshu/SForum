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
                    <div class="row row-cards">
                        @foreach($page as $data)
                            <article class="col-md-12">
                                <div class="card">
                                    <div class="card-header">
                                        <a href="{{$data['url']}}" class="card-title">
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
