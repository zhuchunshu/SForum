@extends("App::app")

@section('title', '「'.$data->name.'」关键词下的帖子列表')
@section('description', '为您展示「'.$data->name.'」关键词下的帖子列表')
@section('keywords', '为您展示「'.$data->name.'」关键词下的帖子列表')

@section('content')
    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-lg-9">
                    @include('Topic::KeyWords.data.index')
                </div>
                <div class="col-lg-3">
                    <div class="row row-cards @if(get_options('theme_right_tool_sticky')!=='true'){{"rd"}}@endif">
                        <div class="col-md-12 @if(get_options('theme_right_tool_sticky')!=='true'){{"sticky"}}@endif" style="top: 105px">
                            @include('Topic::KeyWords.data.right')
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