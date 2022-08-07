@extends("App::app")

@section('title',"修改id为:".$data->id."的评论")


@section('content')

    <div class="row justify-content-center">
        <div class="col-md-12">
            <div class="border-0 card">
                <div class="card-header">
                    <h3 class="card-title"><a class="text-red"
                                              href="/{{$data->topic->id}}.html">{{$data->topic->title}}</a>
                        下id为{{$data->id}}的评论</h3>
                </div>
                <div class="card-body" id="vue-comment-topic-edit-form">
                    <form action="" method="post" @@submit.prevent="submit">
                        <div class="row">
                            <div class="mb-3">
                                <div class="row">
                                    @if(count((new \App\Plugins\Core\src\Lib\Emoji())->get()))
                                        <div class="col-md-3">
                                            <div class="card">
                                                <ul class="nav nav-tabs" data-bs-toggle="tabs" style="flex-wrap: inherit;
        width: 100%;
        height: 3.333333rem;
        padding: 0.373333rem 0.32rem 0;
        box-sizing: border-box;
        /* 下面是实现横向滚动的关键代码 */
        display: inline;
        float: left;
        white-space: nowrap;
        overflow-x: scroll;
        -webkit-overflow-scrolling: touch; /*解决在ios滑动不顺畅问题*/
        overflow-y: hidden;">
                                                    @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                        <li class="nav-item">
                                                            <a href="#emoji-list-{{$key}}"
                                                               class="nav-link @if ($loop->first) active @endif"
                                                               data-bs-toggle="tab">{{$key}}</a>
                                                        </li>
                                                    @endforeach
                                                </ul>
                                                <div class="card-body">
                                                    <div class="tab-content">
                                                        @foreach((new \App\Plugins\Core\src\Lib\Emoji())->get() as $key => $value)
                                                            <div class="tab-pane  @if ($loop->first) active @endif show"
                                                                 id="emoji-list-{{$key}}"
                                                                 style="max-height: 320px;overflow-x: hidden;">
                                                                <div class="row">
                                                                    @if($value['type'] === 'image')
                                                                        @foreach($value['container'] as $emojis)
                                                                            <div @@click="selectEmoji('{{$emojis['text']}}')"
                                                                                 class="col-2 hvr-glow emoji-picker"
                                                                                 emoji-data="{{$emojis['text']}}">{!! $emojis['icon'] !!}</div>
                                                                        @endforeach
                                                                    @endif
                                                                </div>
                                                            </div>
                                                        @endforeach
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="col-md-9">
                                            <div id="vditor"></div>
                                        </div>
                                    @else
                                        <div class="col-md-12">
                                            <div id="vditor"></div>
                                        </div>
                                    @endif
                                    <div class="col-12 mt-3">
                                        <button class="btn btn-primary" style="margin-top: 5px">提交</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
    <script>var comment_id = {{$data->id}}</script>
    <script>var topic_id = {{$data->topic_id}}</script>
    <script src="{{mix("plugins/Comment/js/edit.js")}}"></script>
@endsection