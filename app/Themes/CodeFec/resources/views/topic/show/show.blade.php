@extends("App::app")

@section('title',$data->title)
@section('description',get_content_brief($data->post->content)?:'「关于'.$data->title."」帖子的相关信息")
@section('keywords',$data->title.','.$data->user->username)

@section('content')

    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-lg-9">
                    @foreach(Itf()->get('ui-topic-content-start-hook') as $k=>$v)
                        @if(call_user_func($v['enable'])===true)
                            @include($v['view'])
                        @endif
                    @endforeach
                    @include('App::topic.show.content')
                    @foreach(Itf()->get('ui-topic-content-end-hook') as $k=>$v)
                        @if(call_user_func($v['enable'])===true)
                            @include($v['view'])
                        @endif
                    @endforeach
                </div>
                <div class="col-lg-3">
                    <div class="row row-cards @if(get_options('theme_right_tool_sticky')!=='true'){{"rd"}}@endif">
                        <div class="col-md-12 @if(get_options('theme_right_tool_sticky')!=='true'){{"sticky"}}@endif" style="top: 105px">
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
    <link rel="stylesheet" href="{{ file_hash('css/fancybox.css') }}">
    <link rel="stylesheet" href="{{ file_hash('css/prism.css') }}">
    <link rel="stylesheet" href="{{file_hash('tabler/libs/plyr/dist/plyr.css')}}">
    <style>
        .plyr iframe, .plyr video{
            height: unset;
            max-height: 100%;
        }
    </style>
@endsection

@section('scripts')
    <script>var topic_id ={{$data->id}}</script>
    @if($comment_page)
        <script>var comment_id ={{$comment_page}}</script>
    @endif
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
    <script src="{{mix('plugins/Comment/js/topic.js')}}"></script>
    <script src="{{file_hash('tabler/libs/plyr/dist/plyr.min.js')}}"></script>
    <script src="{{ file_hash('js/prism.js') }}"></script>

    @if(get_options('comment_emoji_close')!=='true')
        <link rel="stylesheet" href="{{file_hash('css/OwO.min.css')}}">
        <script src="{{file_hash('js/OwO.min.js')}}"></script>
        <script>
            if(document.getElementById('create-comment-owo') && document.getElementById('create-comment-textarea')){
                var OwO_demo = new OwO({
                    logo: '[OωO表情]',
                    container: document.getElementById('create-comment-owo'),
                    target: document.getElementById('create-comment-textarea'),
                    api: '/api/core/OwO.json',
                    width: '300px',
                    maxHeight: '250px',
                });
            }
            if(OwO_demo && document.getElementById('create-comment-owo2')){
                var OwO_demo2 = new OwO({
                    logo: '[OωO表情]',
                    container: document.getElementById('create-comment-owo2'),
                    target: document.getElementById('reply-comment-content'),
                    api: '/api/core/OwO.json',
                    width: '300px',
                    maxHeight: '250px',
                });
            }
        </script>
    @endif
    <script>
        document.addEventListener("DOMContentLoaded", function () {
            $("video").css("max-height","400px")
            window.Plyr && (new Plyr('video'));
            (new Plyr('div[data-plyr-provider="youtube"]'));
        });
    </script>
@endsection
