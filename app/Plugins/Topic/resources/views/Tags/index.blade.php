@extends("App::app")

@section('title', __("app.tag"))
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
                            {{__("tag.list")}}
                        </h2>
                    </div>

                    <div class="col-auto">
                        @if(auth()->check() && Authority()->check('topic_tag_create'))
                            <a href="/tags/create" class="btn btn-primary">{{__("tag.create")}}</a>
                        @endif
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')

    <div class="row row-cards">
        @if ($page->count())
            @foreach ($page as $value)
                <div class="col-md-6 col-lg-4">
                    <a href="/tags/{{$value->id}}.html" class="card card-link text-primary-fg" style="background-color: {{$value->color}}!important;">
                        <div class="card-stamp">
                            <div class="card-stamp-icon bg-yellow">
                                <!-- Download SVG icon from http://tabler-icons.io/i/star -->
                                {!! $value->icon !!}
                            </div>
                        </div>
                        <div class="card-body">
                            <h3 class="card-title">{{$value->name}}</h3>
                            <p>{{ \Hyperf\Stringable\Str::limit(core_default($value->description, __("app.no description")), 32) }}</p>
                        </div>
                    </a>
                </div>
            @endforeach
        @else
            <div class="col-md-4">
                <div class="border-0 card">
                    <div class="card-status-top bg-danger"></div>
                    <div class="card-body">
                        <div class="row">
                            <div class="col-auto">
                                <span class="avatar"></span>
                            </div>
                            <div class="col">
                                <h3 class="card-title text-h2">{{__("app.No more results")}}</h3>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        @endif
        {!! make_page($page) !!}
    </div>

@endsection
