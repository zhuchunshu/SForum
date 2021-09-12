@extends("plugins.Core.app")

@section('title', '「'.$data->name.'」关键词下的帖子列表')
@section('description', '为您展示「'.$data->name.'」关键词下的帖子列表')
@section('keywords', '为您展示「'.$data->name.'」关键词下的帖子列表')

@section('content')
    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('plugins.Topic.KeyWords.data.index')
                </div>
                <div class="col-md-5">
                    @include('plugins.Topic.KeyWords.data.right')
                </div>
            </div>
        </div>
    </div>
@endsection


@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection