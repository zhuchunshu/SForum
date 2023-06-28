<div class="row row-cards justify-content-center" id="topic-page">
    @foreach(Itf()->get('ui-start-end-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
    <div class="col-md-12" id="topic">
        <div class="card">
            <div class="card-header">
                <h2 class="card-title text-reset" style="font-size: 1.2rem;line-height: 1.5;" data-bs-toggle="tooltip"
                    data-bs-placement="top" title="{{__('topic.title')}}">
                    {{ $data->title }}
                    @if($data->status==="lock")
                        <span data-bs-toggle="tooltip" data-bs-placement="bottom" title="帖子已锁定" style="display: inline-block" class="text-reset bg-transparent">
                            <svg xmlns="http://www.w3.org/2000/svg"
                                 style="--tblr-icon-size:1.5rem;margin-bottom: 4px"
                                 class="icon icon-tabler icon-tabler-lock" width="20" height="20"
                                 viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                 fill="none" stroke-linecap="round" stroke-linejoin="round">
                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                <path d="M5 11m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z"></path>
                                <path d="M12 16m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"></path>
                                <path d="M8 11v-4a4 4 0 0 1 8 0v4"></path>
                            </svg>
                        </span>
                    @endif
                    @if ($data->essence > 0)
                        <span class="text-green">
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-diamond"
                                 style="--tblr-icon-size:1.8rem" width="24" height="24" viewBox="0 0 24 24"
                                 stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                                 stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M6 5h12l3 5l-8.5 9.5a.7 .7 0 0 1 -1 0l-8.5 -9.5l3 -5"></path>
   <path d="M10 12l-2 -2.2l.6 -1"></path>
</svg>
                        </span>
                    @endif
                </h2>
                <div class="card-actions">
                    @if(auth()->check())
                        <div class="dropdown">
                            <a href="#" class="btn-action dropdown-toggle" data-bs-toggle="dropdown"
                               aria-haspopup="true" aria-expanded="false">
                                <!-- Download SVG icon from http://tabler-icons.io/i/dots-vertical -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"
                                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                     stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                                    <path d="M12 12m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/>
                                    <path d="M12 19m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/>
                                    <path d="M12 5m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"/>
                                </svg>
                            </a>
                            <div class="dropdown-menu dropdown-menu-end text-muted">
                                @foreach(Itf()->get('ui-topic-show-dropdown') as $k=>$v)
                                    @if(call_user_func($v['enable'],$data)===true && $v['view'] instanceof \Closure)
                                        {!! call_user_func($v['view'],$data) !!}
                                    @endif
                                @endforeach
                            </div>
                        </div>
                    @endif
                </div>
            </div>
            @include('App::topic.show.include.author')
            <article class="card-body topic article markdown text-reset">
                {!! ContentParse()->parse($data->post->content,$parseData) !!}
            </article>
            @if($data->user->Options->qianming && $data->user->Options->qianming!=='no bio')
                <div class="px-3 py-3">
                    <div class="hr-text hr-text-left mt-0 mb-3">signature</div>
                    <span class="text-muted">
                            {{$data->user->Options->qianming}}
                    </span>
                </div>
            @endif

            {{--            页脚--}}
            @include('App::topic.show.include.footer')

        </div>
    </div>

    {{--    上下页--}}
    {{--    @include('App::topic.show.include.lfpage')--}}
    @foreach(Itf()->get('ui-topic-comment-before-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
    @if(isset($data->post->options->disable_comment) && $data->post->options->disable_comment)
        <div class="col-md-12">
            <div class="border-0 card">
                <div class="empty">
                    <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-ban" width="24"
                             height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                             stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <circle cx="12" cy="12" r="9"></circle>
                            <line x1="5.7" y1="5.7" x2="18.3" y2="18.3"></line>
                        </svg>
                    </div>
                    <p class="empty-title">禁止评论</p>
                    <p class="empty-subtitle text-muted">
                        此帖子关闭了评论及回复功能
                    </p>
                    @if(!auth()->check())
                        <div class="empty-action">
                            <a href="/login" class="btn btn-primary">
                                <!-- Download SVG icon from http://tabler-icons.io/i/search -->
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login"
                                     width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                     fill="none" stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                    <path d="M14 8v-2a2 2 0 0 0 -2 -2h-7a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h7a2 2 0 0 0 2 -2v-2"></path>
                                    <path d="M20 12h-13l3 -3m0 6l-3 -3"></path>
                                </svg>
                                登陆
                            </a>
                        </div>
                    @endif
                </div>
            </div>
        </div>
    @else
        {{--    显示评论--}}
        @include('Comment::Widget.show-topic')
        @foreach(Itf()->get('ui-topic-comment-after-hook') as $k=>$v)
            @if(call_user_func($v['enable'])===true)
                @include($v['view'])
            @endif
        @endforeach
        {{--    评论--}}
        @foreach(Itf()->get('ui-topic-create-comment-before-hook') as $k=>$v)
            @if(call_user_func($v['enable'])===true)
                @include($v['view'])
            @endif
        @endforeach
        @include('Comment::Widget.topic')
        @foreach(Itf()->get('ui-topic-create-comment-after-hook') as $k=>$v)
            @if(call_user_func($v['enable'])===true)
                @include($v['view'])
            @endif
        @endforeach
    @endif

    @if(auth()->check())
        {{--        举报模态--}}
        @include('App::topic.show.include.report')
    @endif
    @foreach(Itf()->get('ui-topic-end-hook') as $k=>$v)
        @if(call_user_func($v['enable'])===true)
            @include($v['view'])
        @endif
    @endforeach
</div>

