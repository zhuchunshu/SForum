@if($comment_count)
    @if(get_options("comment_topic_show_type","default")==="ajax")
        <div class="col-md-12"  comment-load="topic" topic-id="{{$data->id}}">
            <div class="row row-cards">
                <span class="text-center" comment-load="remove"><h1>正在加载评论<span class="animated-dots"></span></h1></span>
            </div>
        </div>
    @endif
    @if(get_options("comment_topic_show_type","default")==="default")
        @if($comment->count())
            <div class="col-md-12">
                <div class="row row-cards">
                    @foreach($comment as $key=>$value)
                        <div id="comment-{{$value->id}}" name="comment-{{$value->id}}" class="col-md-12">
                            <div class="border-0 card card-body">
                                <div class="row">
{{--                                    作者信息--}}
                                    <div class="col-md-12">
                                        <div class="row">
{{--                                            头像--}}
                                            <div class="col-auto" id="comment-user-avatar-{{$value->id}}" comment-show="user-data" user-id="{{$value->user_id}}">
                                                <a href="/users/{{$value->user->username}}.html"><span class="avatar" style="background-image: url({{super_avatar($value->user)}})"></span></a>
                                            </div>
{{--                                            作者信息--}}
                                            <div class="col text-truncate">
                                                <a href="/users/{{$value->user->username}}.html" class="text-body d-block text-truncate">{{$value->user->username}}</a>
                                                <small data-bs-toggle="tooltip" data-bs-placement="top" data-bs-original-title="{{$value->created_at}}" class="text-muted text-truncate mt-n1">发表于:{{format_date($value->created_at)}}</small>
                                            </div>
{{--                                            楼层信息--}}
                                            <div class="col-auto">
                                                <a href="/{{$data->id}}.html/{{$value->id}}?page={{$comment->currentPage()}}">{{ ($key + 1)+(($comment->currentPage()-1)*10) }}楼</a>
                                            </div>

                                        </div>
                                    </div>
{{--                                    评论内容--}}
                                    <div class="col-md-12">
                                        <div class="hr-text" style="margin-bottom:8px;margin-top:15px">评论内容</div>
                                    </div>
                                    <div class="col-md-12 markdown vditor-reset">
                                        @if($value->parent_id)
                                            <div class="quote">
                                                <blockquote>
                                                    <a style="font-size:13px;" href="{{$value->parent_url}}" target="_blank">
                                                        <span style="color:#999999" >{{$value->parent->user->username}} 发表于 {{$value->created_at}}</span>
                                                    </a>
                                                    <br>
                                                    {{\Hyperf\Utils\Str::limit(remove_bbCode(strip_tags($value->parent->content)),60)}}
                                                </blockquote>
                                            </div>
                                        @endif
                                        {!! ShortCodeR()->handle($value->content) !!}
                                    </div>
{{--                                    操作--}}
                                    <div class="col-md-12">
                                        <div class="hr-text" style="margin-bottom:5px;margin-top:15px">操作</div>
                                    </div>
                                    <div class="col-md-12">
                                        <a style="text-decoration:none;" comment-click="comment-like-topic" comment-id="{{ $value->id }}"
                                           class="cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="点赞">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"
                                                 viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                                                 stroke-linejoin="round">
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                                                <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" />
                                            </svg>
                                            <span comment-show="comment-topic-likes">{{ $value->likes }}</span>
                                        </a>
                                        {{-- markdown --}}
                                        <a data-bs-toggle="tooltip" data-bs-placement="top" href="/comment/topic/{{ $value->id }}.md"
                                           data-bs-original-title="查看markdown文本">
                    <span class="switch-icon-a text-muted">
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-markdown" width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                            <path d="M7 15v-6l2 2l2 -2v6"></path>
                            <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                        </svg>
                    </span>
                                        </a>
                                        <a style="text-decoration:none;" comment-click="comment-reply-topic" comment-id="{{ $value->id }}"
                                           class="cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="回复">
                                            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message-circle-2" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
                                                <line x1="12" y1="12" x2="12" y2="12.01"></line>
                                                <line x1="8" y1="12" x2="8" y2="12.01"></line>
                                                <line x1="16" y1="12" x2="16" y2="12.01"></line>
                                            </svg>
                                        </a>
                                    </div>
                                    <div class="col-md-12" comment-dom="comment-{{$value->id}}" comment-status="off">
                                        <div class="hr-text" style="margin-bottom:15px;margin-top:15px;display: none">回复</div>
                                    </div>
                                    <div class="col-md-12" style="display: none" comment-url="/{{$data->id}}.html/{{$value->id}}?page={{$comment->currentPage()}}" comment-dom="comment-vditor-{{$value->id}}" comment-status="off">
                                        <div id="comment-reply-vditor-{{$value->id}}"></div>
                                        <button style="margin-top:10px" class="btn btn-primary" type="button" comment-dom="comment-vditor-submit-{{$value->id}}">提交</button>
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
