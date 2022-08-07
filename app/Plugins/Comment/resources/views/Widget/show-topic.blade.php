@if($comment_count)
    @if(get_options("comment_topic_show_type","default")==="ajax")
        <div class="col-md-12" comment-load="topic" topic-id="{{$data->id}}">
            <div class="row row-cards">
                <span class="text-center" comment-load="remove"><h1>正在加载评论<span
                                class="animated-dots"></span></h1></span>
            </div>
        </div>
    @endif
    @if(get_options("comment_topic_show_type","default")==="default")
        @php $caina = false; @endphp
        @if($data->user_id == auth()->id() && Authority()->check("comment_caina")) @php $caina = true;@endphp @endif
        @if(Authority()->check("admin_comment_caina")) @php $caina = true; @endphp @endif
        @if($comment->count())
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
                                                    <a href="/users/{{$value->user->username}}.html"><span
                                                                class="avatar"
                                                                style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                                </div>
                                                {{--                                            作者信息--}}
                                                <div class="col text-truncate">
                                                    <a style="white-space:nowrap;"
                                                       href="/users/{{$value->user->username}}.html"
                                                       class="text-body text-truncate">{{$value->user->username}}</a>
                                                    <a data-bs-toggle="tooltip" data-bs-placement="right" title="{{$value->user->class->name}}" href="/users/group/{{$value->user->class->id}}.html" style="color:{{$value->user->class->color}}">
                                                        <span>{!! $value->user->class->icon !!}</span>
                                                    </a>
                                                    @if($value->optimal) <span
                                                            class="badge badge-pill bg-teal">{{__("topic.comment.best reply")}}</span> @endif
                                                    <br/>
                                                    <small data-bs-toggle="tooltip" data-bs-placement="top"
                                                           data-bs-original-title="{{$value->created_at}}"
                                                           class="text-muted text-truncate mt-n1">{{__("app.Published on")}}:{{format_date($value->created_at)}}</small>
                                                    @if(get_options('comment_author_ip','开启')==='开启' && @$value->post->user_ip)
                                                        |
                                                        <small class="text-red" comment-type="ip" comment-id="{{$value->id}}">Loading<span class="animated-dots"></span></small>
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
                                                           data-bs-placement="bottom" title="{{__("topic.comment.adoption")}}"
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
                                        <div class="col-md-12">
                                            <div class="hr-text" style="margin-bottom:8px;margin-top:15px">{{__("topic.comment.comment content")}}</div>
                                        </div>
                                        <div core-show="comment" comment-id="{{$value->id}}"
                                             class="col-md-12 markdown vditor-reset">
                                            @if($value->parent_id)
                                                @if(@$value->parent->id)
                                                    <div class="quote">
                                                        <blockquote>
                                                            <a style="font-size:13px;" href="{{$value->parent_url}}"
                                                               target="_blank">
                                                                <span style="color:#999999">{{$value->parent->user->username}} {{__("app.Published on")}} {{format_date($value->parent->created_at)}}</span>
                                                            </a>
                                                            <br>
                                                            {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($value->parent->post->content)),60) !!}
                                                        </blockquote>
                                                    </div>
                                                @else
                                                    <div class="quote">
                                                        <blockquote>
                                                            引用的评论已被删除
                                                        </blockquote>
                                                    </div>
                                                @endif
                                            @endif
                                            {!!CommentContentParse()->parse($value->post->content,['comment' => $value,'topic' => $data]) !!}
                                        </div>
                                        {{--                                    操作--}}
                                        <div class="col-md-12">
                                            <div class="hr-text" style="margin-bottom:5px;margin-top:15px">{{__("topic.comment.operate")}}</div>
                                        </div>
                                        <div class="col-md-12">
                                            {{--                                            点赞--}}
                                            <a style="text-decoration:none;" comment-click="comment-like-topic"
                                               comment-id="{{ $value->id }}"
                                               class="cursor-pointer text-muted hvr-icon-bounce"
                                               data-bs-toggle="tooltip" data-bs-placement="bottom" title="{{__("topic.likes")}}">
                                                <svg xmlns="http://www.w3.org/2000/svg" class="icon hvr-icon" width="24"
                                                     height="24"
                                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                                     fill="none" stroke-linecap="round"
                                                     stroke-linejoin="round">
                                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                                    <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572"/>
                                                </svg>
                                                <span comment-show="comment-topic-likes">{{ $value->likes }}</span>
                                            </a>
                                            {{-- markdown --}}
                                            <a style="text-decoration:none;" data-bs-toggle="tooltip"
                                               data-bs-placement="top" href="/comment/topic/{{ $value->id }}.md"
                                               data-bs-original-title="{{__("app.preview markdown")}}" class="hvr-icon-grow-rotate">
                    <span class="switch-icon-a text-muted">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-markdown hvr-icon"
                             width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                            <path d="M7 15v-6l2 2l2 -2v6"></path>
                            <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                        </svg>
                    </span>
                                            </a>
                                            {{--                                            回复--}}
                                            <a style="text-decoration:none;" comment-click="comment-reply-topic"
                                               comment-id="{{ $value->id }}"
                                               class="cursor-pointer text-muted hvr-icon-up" data-bs-toggle="tooltip"
                                               data-bs-placement="bottom" title="{{__("app.reply")}}">
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
                                            </a>


                                            {{--                                        删除评论--}}
                                            @if(auth()->check())

                                                <a style="text-decoration:none;" comment-click="comment-delete-topic"
                                                   comment-id="{{ $value->id }}"
                                                   class="cursor-pointer text-muted hvr-icon-drop"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom" title="{{__("app.delete")}}">
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
                                                </a>
                                            @endif
                                            {{--                                        修改评论--}}
                                            @if(auth()->check())
                                                <a style="text-decoration:none;"
                                                   href="/comment/topic/{{$value->id}}/edit"
                                                   class="hvr-icon-fade cursor-pointer text-muted"
                                                   data-bs-toggle="tooltip" data-bs-placement="bottom" title="编辑">
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
                                                </a>
                                            @endif
                                            {{--                                            引用评论--}}
                                            <a style="text-decoration:none;"  core-click="copy" copy-content="[comment comment_id={{$value->id}}]" message="短代码复制成功!"
                                               class="cursor-pointer text-muted hvr-icon-up" data-bs-toggle="tooltip"
                                               data-bs-placement="bottom" title="引用">
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
                                            </a>
                                        </div>

                                        <div class="col-md-12" comment-dom="comment-{{$value->id}}"
                                             comment-status="off">
                                            <div class="hr-text"
                                                 style="margin-bottom:15px;margin-top:15px;display: none">{{__("app.reply")}}
                                            </div>
                                        </div>
                                        <div class="col-md-12" style="display: none"
                                             comment-url="/{{$data->id}}.html/{{$value->id}}?page={{$comment->currentPage()}}"
                                             comment-dom="comment-vditor-{{$value->id}}" comment-status="off">
                                            <div id="comment-reply-vditor-{{$value->id}}"></div>
                                            <button style="margin-top:10px" class="btn btn-primary" type="button"
                                                    comment-dom="comment-vditor-submit-{{$value->id}}">提交
                                            </button>
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
