<div class="row row-cards justify-content-center">
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
                                        <a href="/users/{{$value->user->username}}.html"><span class="avatar" style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                    </div>
                                    {{--                                            作者信息--}}
                                    <div class="col text-truncate">
                                        <a style="white-space:nowrap;" href="/users/{{$value->user->username}}.html" class="text-body text-truncate">{{$value->user->username}}</a>
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
                            <div class="col-md-12">
                                <div class="hr-text" style="margin-bottom:8px;margin-top:15px">{{__("topic.comment.comment content")}}</div>
                            </div>
                            <div core-show="comment" comment-id="{{$value->id}}" class="col-md-12 markdown vditor-reset">
                                @if($value->parent_id)
                                    <div class="quote">
                                        <blockquote>
                                            <a style="font-size:13px;" href="{{$value->parent_url}}" target="_blank">
                                                <span style="color:#999999" >{{$value->parent->user->username}} {{__("app.Published on")}} {{format_date($value->created_at)}}</span>
                                            </a>
                                            <br>
                                            {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($value->parent->content)),60) !!}
                                        </blockquote>
                                    </div>
                                @endif
                                {!! $value->content !!}
                            </div>
                            {{--                                    操作--}}
                            <div class="col-md-12">
                                <div class="hr-text" style="margin-bottom:5px;margin-top:15px">{{__("topic.comment.operate")}}</div>
                            </div>
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
                                    <span comment-show="comment-topic-likes">{{ $value->likes }}</span>
                                </a>
                                {{-- markdown --}}
                                <a style="text-decoration:none;" data-bs-toggle="tooltip" data-bs-placement="top" href="/comment/topic/{{ $value->id }}.md"
                                   data-bs-original-title="{{__("app.preview markdown")}}" class="hvr-icon-grow-rotate">
                    <span class="switch-icon-a text-muted">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-markdown hvr-icon" width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                            <path d="M7 15v-6l2 2l2 -2v6"></path>
                            <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                        </svg>
                    </span>
                                </a>

                                {{--                                        收藏--}}
                                @if(auth()->check())

                                    <a style="text-decoration:none;" topic-id="{{$value->topic_id}}" comment-click="star-comment" comment-id="{{ $value->id }}"
                                       class="cursor-pointer text-muted hvr-icon-up" data-bs-toggle="tooltip" data-bs-placement="bottom" title="收藏">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-star" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                                        </svg>
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