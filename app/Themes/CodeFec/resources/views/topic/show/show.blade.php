@extends("App::app")

@section('title',$data->title)
@section('description',content_brief($data->post->content,get_options("topic_brief_len",250)))
@section('keywords',$data->title.','.$data->user->username.','.content_brief($data->post->content,get_options("topic_brief_len",250)))

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-md-9">
                    @include('App::topic.show.content')
                </div>
                <div class="col-md-3">
                    <div class="row row-cards rd">
                        <div class="col-md-12 sticky" style="top: 105px">
                            @include('App::topic.show.show-right')
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
    <link rel="stylesheet" href="{{file_hash('tabler/libs/plyr/dist/plyr.css')}}">
@endsection

@section('scripts')
    <script>var topic_id={{$data->id}}</script>
    @if($comment_page)
        <script>var comment_id={{$comment_page}}</script>
    @endif
    <script src="{{ mix('plugins/Topic/js/topic.js') }}"></script>
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
    <script src="{{mix('plugins/Comment/js/topic.js')}}"></script>
    <script src="{{file_hash('tabler/libs/plyr/dist/plyr.min.js')}}"></script>
    <script>
        document.addEventListener("DOMContentLoaded", function () {
            window.Plyr && (new Plyr('video'));
        });
    </script>
@endsection
