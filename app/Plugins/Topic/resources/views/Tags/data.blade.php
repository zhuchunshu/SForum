@extends("plugins.Core.app")

@section('title', '「'.$data->name.'」标签的相关信息,以及此标签下的所有帖子')
@section('description', '为您展示「'.$data->name.'」标签的相关信息,以及此标签下的所有帖子')
@section('keywords', '为您展示「'.$data->name.'」标签的相关信息,以及此标签下的所有帖子')

@section('content')
    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('plugins.Topic.Tags.data.index')
                </div>
                <div class="col-md-5">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('plugins.Topic.Tags.data.right')
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