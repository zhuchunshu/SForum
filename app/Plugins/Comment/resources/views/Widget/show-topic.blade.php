@if($comment->total())
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
                                    <div class="mt-3 d-flex flex-lg-row-reverse">
                                        <div class="OwO" id="create-comment-owo2">[OωO表情]</div>
                                    </div>
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
                <div class="card">

                    <div class="card-header">
                        <h3 class="card-title">全部评论</h3>
                        <div class="card-actions">
                            @if($comment_sort=="desc")
                                <a href="?{{ core_http_build_query(request()->all(),['comment_sort' => 'asc'])  }}">正序显示↑</a>
                            @else
                                <a href="?{{ core_http_build_query(request()->all(),['comment_sort' => 'desc'])  }}">倒序显示↓</a>
                            @endif
                        </div>
                    </div>

                    <div class="card-body">

                        <div class="row row-cards">
                            @foreach($comment as $key=>$value)
                                @if(!$loop->first)<div class="hr-text mt-0 mb-0">next</div>@endif
                                <div id="comment-{{$value->id}}" name="comment-{{$value->id}}" class="col-md-12 mt-1 @if($comment_page==$value->id) bg-cyan-lt @endif">
                                    <div class="@if($value->optimal)comment-optimal @endif">
                                        <div class="mt-2">
                                            <div class="row">
                                                {{--                                    作者信息--}}
                                                <div class="col-md-12">
                                                    <div class="row">
                                                        {{--                                            头像--}}
                                                        <div class="col-auto" id="comment-user-avatar-{{$value->id}}"
                                                             comment-show="user-data" user-id="{{$value->user_id}}">
                                                            <a href="/users/{{$value->user->id}}.html"><span
                                                                        class="avatar avatar-rounded"
                                                                        style="background-image: url({{super_avatar($value->user)}})">

                                                        </span></a>
                                                        </div>
                                                        {{--                                            作者信息--}}
                                                        <div class="col text-truncate my-0" style="margin-left: -10px">
                                                            {!! u_username($value->user,['extends' => true,'comment' => true,'class' =>['text-body','text-truncate'],'style' => 'white-space:nowrap;']) !!}
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
                                                @if(auth()->check())
                                                    @include('Comment::shared.footer_tool')
                                                @endif
                                            </div>
                                        </div>

                                    </div>
                                </div>
                            @endforeach
                        </div>

                    </div>

                    @if($comment->hasPages())
                        <div class="card-footer pb-0">
                            {!! make_page($comment) !!}
                        </div>
                    @endif


                </div>
            </div>
        @endif
    @endif
@endif
