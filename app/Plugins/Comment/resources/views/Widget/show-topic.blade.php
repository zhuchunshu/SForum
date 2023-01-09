@if($comment_count)
    @if(get_options("comment_topic_show_type","default")==="default")
        @php $caina = false; @endphp
        @if($data->user_id == auth()->id() && Authority()->check("comment_caina")) @php $caina = true;@endphp @endif
        @if(Authority()->check("admin_comment_caina")) @php $caina = true; @endphp @endif
        @if($comment->count())
            @if(@isset($data->post->options->only_author) && $data->post->options->only_author)
                @php($posts_options_only_author = true)
            @else
                @php($posts_options_only_author = false)
            @endif
            <div class="modal modal-blur fade" id="reply-comment-modal" tabindex="-1" role="dialog" aria-hidden="true">
                <div class="modal-dialog modal-dialog-centered" role="document">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">回复评论</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            @if(authManager()->guest())
                                <div class="empty">
                                    <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"></path>
                                            <path d="M20 12h-13l3 -3m0 6l-3 -3"></path>
                                        </svg>
                                    </div>
                                    <p class="empty-title">无权限</p>
                                    <p class="empty-subtitle text-muted">
                                        请登录后评论
                                    </p>
                                </div>
                            @else
                                <div>
                                    <label class="form-label">回复内容</label>
                                    <input type="hidden" name="comment_id" value="" id="reply-comment-id">
                                    <textarea class="form-control" name="content" id="reply-comment-content" data-bs-toggle="autosize" placeholder="说点什么..."></textarea>
                                </div>
                            @endif
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn me-auto" data-bs-dismiss="modal">关闭</button>
                            @if(auth()->check())
                                <button type="button" class="btn btn-primary" id="reply-comment-modal-reply-button" data-bs-dismiss="modal">回复</button>
                            @else
                                <a href="/login" class="btn btn-primary">
                                    <!-- Download SVG icon from http://tabler-icons.io/i/search -->
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                        <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"></path>
                                        <path d="M20 12h-13l3 -3m0 6l-3 -3"></path>
                                    </svg>
                                    登陆
                                </a>
                            @endif
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-12">
                <div class="row row-cards">
                    @foreach($comment as $key=>$value)
                        <div id="comment-{{$value->id}}" name="comment-{{$value->id}}" class="col-md-12">
                            <div class="card @if($value->optimal) comment-optimal @else border-0 @endif">
                                <div class="card-body">
                                    <div class="row">
                                        {{--                                    作者信息--}}
                                        <div class="col-md-12">
                                            <div class="row">
                                                {{--                                            头像--}}
                                                <div class="col-auto" id="comment-user-avatar-{{$value->id}}"
                                                     comment-show="user-data" user-id="{{$value->user_id}}">
                                                    <a href="/users/{{$value->user->id}}.html"><span
                                                                class="avatar"
                                                                style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                                </div>
                                                {{--                                            作者信息--}}
                                                <div class="col text-truncate">
                                                    <a style="white-space:nowrap;"
                                                       href="/users/{{$value->user->id}}.html"
                                                       class="text-body text-truncate">{{$value->user->username}}</a>
                                                    <a data-bs-toggle="tooltip" data-bs-placement="right"
                                                       title="{{$value->user->class->name}}"
                                                       href="/users/group/{{$value->user->class->id}}.html"
                                                       style="color:{{$value->user->class->color}}">
                                                        <span>{!! $value->user->class->icon !!}</span>
                                                    </a>
                                                    @if($value->optimal) <span
                                                            class="badge badge-pill bg-teal">{{__("topic.comment.best reply")}}</span> @endif
                                                    <br/>
                                                    <small data-bs-toggle="tooltip" data-bs-placement="top"
                                                           data-bs-original-title="{{$value->created_at}}"
                                                           class="text-muted text-truncate mt-n1">{{__("app.Published on")}}
                                                        :{{format_date($value->created_at)}}</small>
                                                    @if(get_options('comment_author_ip','开启')==='开启' && @$value->post->user_ip)
                                                        |
                                                        <small class="text-red" comment-type="ip"
                                                               comment-id="{{$value->id}}">Loading<span
                                                                    class="animated-dots"></span></small>
                                                    @endif
                                                </div>
                                                {{--                                            楼层信息--}}
                                                <div class="col-auto">
                                                    <a href="/{{$data->id}}.html/{{$value->id}}?page={{$comment->currentPage()}}">
                                                        {{__("topic.floor",['floor' => ($key + 1)+(($comment->currentPage()-1)*get_options('comment_page_count',15)) ])}}</a>
                                                    @if($caina)
                                                        ·
                                                        <a style="text-decoration:none;"
                                                           comment-click="comment-caina-topic"
                                                           comment-id="{{ $value->id }}" data-bs-toggle="tooltip"
                                                           data-bs-placement="bottom"
                                                           title="{{__("topic.comment.adoption")}}"
                                                           class="cursor-pointer text-teal">
                                                            @if($value->optimal)
                                                                {{__("topic.comment.cancel")}}
                                                            @else
                                                                {{__("topic.comment.adoption")}}
                                                            @endif
                                                        </a>
                                                    @endif
                                                </div>

                                            </div>
                                        </div>
                                        {{--                                    评论内容--}}
{{--                                        <div class="col-md-12">--}}
{{--                                            <div class="hr-text"--}}
{{--                                                 style="margin-bottom:8px;margin-top:15px">{{__("topic.comment.comment content")}}</div>--}}
{{--                                        </div>--}}
                                        @if($posts_options_only_author && auth()->id()!=$value->user_id && auth()->id()!=$data->user_id)
                                            @include('Comment::Widget.only-author')
                                        @else
                                            @include('Comment::Widget.source')
                                        @endif
                                        {{--                                    操作--}}
{{--                                        <div class="col-md-12">--}}
{{--                                            <div class="hr-text"--}}
{{--                                                 style="margin-bottom:5px;margin-top:15px">{{__("topic.comment.operate")}}</div>--}}
{{--                                        </div>--}}
                                        <div class="col-md-12">
                                            {{--                                            点赞--}}
                                            <a style="text-decoration:none;" comment-click="comment-like-topic"
                                               comment-id="{{ $value->id }}"
                                               class="cursor-pointer text-muted hvr-icon-bounce"
                                               data-bs-toggle="tooltip" data-bs-placement="bottom"
                                               title="{{__("topic.likes")}}">
                                                <svg xmlns="http://www.w3.org/2000/svg" class="icon hvr-icon" width="24"
                                                     height="24"
                                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                                     fill="none" stroke-linecap="round"
                                                     stroke-linejoin="round">
                                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                                    <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572"/>
                                                </svg>
                                                <span comment-show="comment-topic-likes">{{ $value->likes->count() }}</span>
                                            </a>
                                            {{--                                            回复--}}
                                            <a style="text-decoration:none;" comment-click="comment-reply-topic"
                                               comment-id="{{ $value->id }}" data-bs-toggle="modal" data-bs-target="#reply-comment-modal"
                                               class="cursor-pointer text-muted hvr-icon-up" title="{{__("app.reply")}}">
                                                <svg xmlns="http://www.w3.org/2000/svg"
                                                     class="hvr-icon icon icon-tabler icon-tabler-message-circle-2"
                                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2"
                                                     stroke="currentColor" fill="none" stroke-linecap="round"
                                                     stroke-linejoin="round">
                                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                    <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
                                                    <line x1="12" y1="12" x2="12" y2="12.01"></line>
                                                    <line x1="8" y1="12" x2="8" y2="12.01"></line>
                                                    <line x1="16" y1="12" x2="16" y2="12.01"></line>
                                                </svg>
                                                {{__("app.reply")}}
                                            </a>


                                            {{--                                        删除评论--}}
                                            @if(auth()->check())

                                                <a style="text-decoration:none;" comment-click="comment-delete-topic"
                                                   comment-id="{{ $value->id }}"
                                                   class="cursor-pointer text-muted hvr-icon-drop"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom"
                                                   title="{{__("app.delete")}}">
                                                    <svg xmlns="http://www.w3.org/2000/svg"
                                                         class="hvr-icon icon icon-tabler icon-tabler-trash" width="24"
                                                         height="24" viewBox="0 0 24 24" stroke-width="2"
                                                         stroke="currentColor" fill="none" stroke-linecap="round"
                                                         stroke-linejoin="round">
                                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                        <line x1="4" y1="7" x2="20" y2="7"></line>
                                                        <line x1="10" y1="11" x2="10" y2="17"></line>
                                                        <line x1="14" y1="11" x2="14" y2="17"></line>
                                                        <path d="M5 7l1 12a2 2 0 0 0 2 2h8a2 2 0 0 0 2 -2l1 -12"></path>
                                                        <path d="M9 7v-3a1 1 0 0 1 1 -1h4a1 1 0 0 1 1 1v3"></path>
                                                    </svg>
                                                    {{__("app.delete")}}
                                                </a>
                                            @endif
                                            {{--                                        修改评论--}}
                                            @if(auth()->check())
                                                <a style="text-decoration:none;"
                                                   href="/comment/topic/{{$value->id}}/edit"
                                                   class="hvr-icon-fade cursor-pointer text-muted"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom" title="修改">
                                                    <svg xmlns="http://www.w3.org/2000/svg"
                                                         class="hvr-icon icon icon-tabler icon-tabler-edit" width="24"
                                                         height="24" viewBox="0 0 24 24" stroke-width="2"
                                                         stroke="currentColor" fill="none" stroke-linecap="round"
                                                         stroke-linejoin="round">
                                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                        <path d="M9 7h-3a2 2 0 0 0 -2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2 -2v-3"></path>
                                                        <path d="M9 15h3l8.5 -8.5a1.5 1.5 0 0 0 -3 -3l-8.5 8.5v3"></path>
                                                        <line x1="16" y1="5" x2="19" y2="8"></line>
                                                    </svg>
                                                    修改
                                                </a>

                                                <a style="text-decoration:none;" topic-id="{{$data->id}}"
                                                   comment-click="star-comment" comment-id="{{ $value->id }}"
                                                   class="cursor-pointer text-muted hvr-icon-up"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom" title="收藏">
                                                    <svg xmlns="http://www.w3.org/2000/svg"
                                                         class="hvr-icon icon icon-tabler icon-tabler-star" width="24"
                                                         height="24" viewBox="0 0 24 24" stroke-width="2"
                                                         stroke="currentColor" fill="none" stroke-linecap="round"
                                                         stroke-linejoin="round">
                                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                        <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                                                    </svg>
                                                    收藏
                                                </a>
                                                {{--    举报--}}
                                                <a data-bs-toggle="modal" data-bs-target="#modal-report"
                                                   url="/{{$data->id}}.html/{{$value->id}}?page={{$comment->currentPage()}}"
                                                   style="text-decoration:none;" topic-id="{{$data->id}}"
                                                   comment-click="report-comment" comment-id="{{ $value->id }}"
                                                   class="cursor-pointer text-muted hvr-icon-pulse"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom" title="举报">
                                                    <svg xmlns="http://www.w3.org/2000/svg"
                                                         class="hvr-icon icon icon-tabler icon-tabler-flag-3" width="24"
                                                         height="24" viewBox="0 0 24 24" stroke-width="2"
                                                         stroke="currentColor" fill="none" stroke-linecap="round"
                                                         stroke-linejoin="round">
                                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                        <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
                                                    </svg>
                                                    举报
                                                </a>
                                            @endif
                                            {{--                                            引用评论--}}
                                            <a style="text-decoration:none;" core-click="copy"
                                               copy-content="[comment comment_id={{$value->id}}]" message="短代码复制成功!"
                                               class="cursor-pointer text-muted hvr-icon-up" data-bs-toggle="tooltip"
                                               data-bs-placement="bottom" title="短代码">
                                                <svg xmlns="http://www.w3.org/2000/svg"
                                                     class="icon icon-tabler icon-tabler-blockquote" width="24"
                                                     height="24" viewBox="0 0 24 24" stroke-width="2"
                                                     stroke="currentColor" fill="none" stroke-linecap="round"
                                                     stroke-linejoin="round">
                                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                    <path d="M6 15h15"></path>
                                                    <path d="M21 19h-15"></path>
                                                    <path d="M15 11h6"></path>
                                                    <path d="M21 7h-6"></path>
                                                    <path d="M9 9h1a1 1 0 1 1 -1 1v-2.5a2 2 0 0 1 2 -2"></path>
                                                    <path d="M3 9h1a1 1 0 1 1 -1 1v-2.5a2 2 0 0 1 2 -2"></path>
                                                </svg>
                                                短代码
                                            </a>
                                        </div>
                                    </div>
                                </div>

                            </div>
                        </div>
                    @endforeach
                </div>
            </div>
            <div>
                {!! make_page($comment) !!}
            </div>
        @endif
    @endif
@endif
