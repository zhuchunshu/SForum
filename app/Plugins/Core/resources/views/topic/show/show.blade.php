@extends("Core::app")

@section('title',$data->title)
@section('description','为您展示:'.$data->title."帖子信息")
@section('keywords',$data->title.','.$data->user->username)

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-10">
            <div class="row row-cards justify-content-center">
                <div class="col-md-7">
                    @include('Core::topic.show.content')
                </div>
                <div class="col-md-5">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('Core::topic.show.show-right')
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
    <script>var topic_id={{$data->id}}</script>
    @if($comment_page)
        <script>var comment_id={{$comment_page}}</script>
    @endif
    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
    <script src="{{mix('plugins/Comment/js/topic.js')}}"></script>
@endsection
