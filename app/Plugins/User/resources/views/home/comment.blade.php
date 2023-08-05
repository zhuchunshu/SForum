<link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
<div class="row row-cards">

    @php($topics = \App\Plugins\Comment\src\Model\TopicComment::query()->where(['user_id' => $user->id,])->orderByDesc('id')->paginate(15))
    <div class="col-md-12">
        <div class="border-0 card card-body">
            <h3 class="card-title">{{$user->username}} 发布的评论</h3>
            @if($topics->count())
                <div class="row row-cards">
                    @foreach($topics as $value)
                        <div class="col-md-12">
                            <div class="card">
                                <div id="comment-{{$value->id}}" name="comment-{{$value->id}}" class="col-md-12">
                                    <div class="card @if($value->optimal) comment-optimal @else border-0 @endif">
                                        <div class="card-body">
                                            <div class="row">
                                                {{--                                    作者信息--}}
                                                <div class="col-md-12">
                                                    <div class="row">
                                                        {{--                                            头像--}}
                                                        <div class="col-auto" id="comment-user-avatar-{{$value->id}}" comment-show="user-data" user-id="{{$user->id}}">
                                                            <a href="/users/{{$user->id}}.html"><span class="avatar" style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                                        </div>
                                                        {{--                                            作者信息--}}
                                                        <div class="col text-truncate">
                                                            <a style="white-space:nowrap;" href="/users/{{$user->id}}.html" class="text-body text-truncate">{{$user->username}}</a>
                                                            <br />
                                                            <small data-bs-toggle="tooltip" data-bs-placement="top" data-bs-original-title="{{$value->created_at}}" class="text-muted text-truncate mt-n1">{{__("app.Published on")}}:{{format_date($value->created_at)}}</small>
                                                        </div>
                                                        {{--                                            楼层信息--}}
                                                        <div class="col-auto">
                                                            <a class="badge badge-pill bg-teal" href="{{get_topic_comment_url($value->id)}}">查看评论</a>
                                                        </div>

                                                    </div>
                                                </div>
                                                {{--                                    评论内容--}}
{{--                                                <div class="col-md-12">--}}
{{--                                                    <div class="hr-text" style="margin-bottom:8px;margin-top:15px">{{__("topic.comment.comment content")}}</div>--}}
{{--                                                </div>--}}
                                                <div core-show="comment" comment-id="{{$value->id}}" class="col-md-12 mt-2 mb-2 markdown">
                                                    @if($value->parent_id)
                                                        <div class="quote">
                                                            <blockquote>
                                                                <a style="font-size:13px;" href="{{$value->parent_url}}" target="_blank">
                                                                    <span style="color:#999999" >{{$value->parent->user->username}} {{__("app.Published on")}} {{format_date($value->created_at)}}</span>
                                                                </a>
                                                                <br>
                                                                {!! \Hyperf\Stringable\Str::limit(remove_bbCode(strip_tags($value->parent->post->content)),60) !!}
                                                            </blockquote>
                                                        </div>
                                                    @endif
                                                    {!!CommentContentParse()->parse($value->post->content,['comment' => $value,'topic' => $value->topic,'remove_shortCode' => ['topic-comment']]) !!}
                                                </div>
                                                {{--                                    操作--}}
{{--                                                <div class="col-md-12">--}}
{{--                                                    <div class="hr-text" style="margin-bottom:5px;margin-top:15px">{{__("topic.comment.operate")}}</div>--}}
{{--                                                </div>--}}

                                            </div>
                                        </div>

                                    </div>
                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
                <div class="mt-2">
                    {!! make_page($topics) !!}
                </div>
            @else
                <div class="empty">
                    <div class="empty-header">404</div>
                    <p class="empty-title">暂无更多结果</p>
                </div>
            @endif
        </div>
    </div>
</div>