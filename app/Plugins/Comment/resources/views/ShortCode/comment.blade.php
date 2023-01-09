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
                                        <a class="badge badge-pill bg-teal" href="/{{$value->topic_id}}.html">访问所在帖子</a>
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
                                                {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($value->parent->post->content)),60) !!}
                                            </blockquote>
                                        </div>
                                    @endif
                                    {!!CommentContentParse()->parse($value->post->content,['comment' => $value,'topic' => $value->topic,'RemoveshortCode' => ['topic-comment']]) !!}
                                </div>
                            @endif
                            {{--                                    操作--}}
{{--                            <div class="col-md-12">--}}
{{--                                <div class="hr-text" style="margin-bottom:5px;margin-top:15px">{{__("topic.comment.operate")}}</div>--}}
{{--                            </div>--}}
                            <div class="col-md-12">
                                {{--                                            点赞--}}
                                <a style="text-decoration:none;" comment-click="comment-like-topic" comment-id="{{ $value->id }}"
                                   class="cursor-pointer text-muted hvr-icon-bounce" data-bs-toggle="tooltip" data-bs-placement="bottom" title="{{__("topic.likes")}}">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon hvr-icon" width="24" height="24"
                                         viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                                         stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                                        <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" />
                                    </svg>
                                    <span comment-show="comment-topic-likes">{{ $value->likes->count() }}</span>
                                </a>

                                {{--                                        收藏--}}
                                @if(auth()->check())

                                    <a style="text-decoration:none;" topic-id="{{$value->topic_id}}" comment-click="star-comment" comment-id="{{ $value->id }}"
                                       class="cursor-pointer text-muted hvr-icon-up" data-bs-toggle="tooltip" data-bs-placement="bottom" title="收藏">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-star" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                                        </svg>
                                        收藏
                                    </a>

                                @endif
                            </div>

                        </div>
                    </div>

                </div>
            </div>
        </div>
    </div>
</div>