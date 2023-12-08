<div class="row row-cards justify-content-center">
    @if(@isset($value->topic->post->options->only_author) && $value->topic->post->options->only_author)
        @php($posts_options_only_author = true)
    @else
        @php($posts_options_only_author = false)
    @endif
    <div class="col-md-10">
        <div class="card">
            <div id="comment-{{$value->id}}" name="comment-{{$value->id}}" class="col-md-12">
                <div class="card @if($value->optimal) comment-optimal @else border-0 @endif">
                    <div class="card-body">
                        <div class="row">
                            {{--                                    作者信息--}}
                            <div class="col-md-12">
                                <div class="row">
                                    {{--                                            头像--}}
                                    <div class="col-auto" id="comment-user-avatar-{{$value->id}}" comment-show="user-data" user-id="{{$value->user_id}}">
                                        <a href="/users/{{$value->user->id}}.html"><span class="avatar" style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                    </div>
                                    {{--                                            作者信息--}}
                                    <div class="col text-truncate">
                                        <a style="white-space:nowrap;" href="/users/{{$value->user->id}}.html" class="text-body text-truncate">{{$value->user->username}}</a>
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
{{--                            <div class="col-md-12">--}}
{{--                                <div class="hr-text" style="margin-bottom:8px;margin-top:15px">{{__("topic.comment.comment content")}}</div>--}}
{{--                            </div>--}}
                            @if($posts_options_only_author)
                                @include('Comment::Widget.only-author')
                            @else
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
                                    {!! $value->post->content !!}
                                </div>
                            @endif

                        </div>
                    </div>

                </div>
            </div>
        </div>
    </div>
</div>