@extends("Core::app")

@section('title',$data->title)
@section('description','为您展示:'.$data->title."帖子信息")
@section('keywords',$data->title.','.$data->user->username)

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('Core::topic.content')
                </div>
                <div class="col-md-5">
                    @include('Core::topic.show-right')
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection

@section('scripts')
    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
@endsection