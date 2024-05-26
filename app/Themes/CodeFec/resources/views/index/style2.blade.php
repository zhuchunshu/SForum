<article class="col-md-12 p-3 pt-2 pb-2 @if($page->hasPages()) border-bottom @endif">
    <div class="d-flex border-0 card">
        <div class="row">
            <div class="col-auto align-self-center">
                <a href="/users/{{$data->user->id}}.html" class="avatar"
                   style="background-image: url({{super_avatar($data->user)}});--tblr-avatar-size: 2.8rem;">
                </a>

            </div>
            <div class="col">
                <div class="row">
                    <div class="col-md-12">
                        <div class="row">
                            <div class="col-md-12 markdown home-article">
                                <h3 class="text-muted @if(get_options('theme_home_title_truncate')==="true"){{"text-truncate"}}@endif">
                                    @if($data->topping>0)
                                        <span class="badge bg-red">
                                                    {{__('app.top')}}
                                                </span>
                                    @else
                                        <span class="badge d-none d-lg-inline-block"
                                              style="background-color: {{$data->tag->color}}!important;">
                                                        {{$data->tag->name}}
                                                    </span>
                                        @if(get_options('theme_home_tag_icon')==='true')
                                            <span class="badge d-inline-block d-lg-none"
                                                  style="background-color: {{$data->tag->color}}!important;">
                                                        {!! $data->tag->icon !!}
                                                    </span>
                                        @endif

                                    @endif
                                    @if($data->essence>0)
                                        <span class="badge bg-green d-none d-lg-inline-block">
                                                        {{__("app.essence")}}
                                                    </span>
                                    @endif

                                    <a href="/{{$data->id}}.html" class="text-reset">
                                        {{$data->title}}
                                    </a>
                                    @if($data->status==="lock")
                                        <div data-bs-toggle="tooltip" data-bs-placement="top" title="帖子已锁定"
                                             style="display: inline-block" class="text-reset bg-transparent">
                                            <svg xmlns="http://www.w3.org/2000/svg"
                                                 style="margin-bottom: 3px"
                                                 class="icon icon-tabler icon-tabler-lock" width="20" height="20"
                                                 viewBox="0 0 24 24" stroke-width="2" stroke="currentColor"
                                                 fill="none" stroke-linecap="round" stroke-linejoin="round">
                                                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                                <path d="M5 11m0 2a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v6a2 2 0 0 1 -2 2h-10a2 2 0 0 1 -2 -2z"></path>
                                                <path d="M12 16m-1 0a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"></path>
                                                <path d="M8 11v-4a4 4 0 0 1 8 0v4"></path>
                                            </svg>
                                        </div>
                                    @endif
                                </h3>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-12">
                        <div class="d-flex align-items-center">
                            <div class="text-muted-sm" style="margin-top:5px">
                                {!! u_username($data->user,['class' => ['text-muted-sm']]) !!} {{format_date($data->created_at)}}

                                @if($data->comments->count())
                                    ←
                                    {{--                                最后回复--}}
                                    {!! u_username($data->comments->last()->user,['class' => ['text-muted-sm']]) !!}  {{format_date($data->comments->last()->created_at)}}

                                    {{--                                    @if($data->updated_at>$data->comments->last()->created_at)--}}
                                    {{--                                        {!! u_username($data->user,['class' => ['text-muted-sm']]) !!} {{format_date($data->updated_at)}}--}}
                                    {{--                                    @else--}}
                                    {{--                                        {!! u_username($data->comments->last()->user,['class' => ['text-muted-sm']]) !!}  {{format_date($data->comments->last()->created_at)}}--}}
                                    {{--                                    @endif--}}

                                @endif
                            </div>
                            <div class="ms-auto d-none d-lg-inline-block">
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom"
                                          title="{{__("app.pageviews")}}">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"
                                             viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                             stroke-linecap="round" stroke-linejoin="round"><path stroke="none"
                                                                                                  d="M0 0h24v24H0z"
                                                                                                  fill="none"/><circle
                                                    cx="12" cy="12" r="2"/><path
                                                    d="M22 12c-2.667 4.667 -6 7 -10 7s-7.333 -2.333 -10 -7c2.667 -4.667 6 -7 10 -7s7.333 2.333 10 7"/></svg>
                                        {{$data->view}}
                                    </span>
                                <a style="text-decoration:none;" href="/{{$data->id}}.html#topic-comment"
                                   class="ms-3 text-muted cursor-pointer" data-bs-toggle="tooltip"
                                   data-bs-placement="bottom" title="{{__("app.comment")}}">
                                    <svg xmlns="http://www.w3.org/2000/svg"
                                         class="icon icon-tabler icon-tabler-message-circle" width="24" height="24"
                                         viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                                         stroke-linecap="round" stroke-linejoin="round">
                                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                        <path d="M3 20l1.3 -3.9a9 8 0 1 1 3.4 2.9l-4.7 1"></path>
                                        <line x1="12" y1="12" x2="12" y2="12.01"></line>
                                        <line x1="8" y1="12" x2="8" y2="12.01"></line>
                                        <line x1="16" y1="12" x2="16" y2="12.01"></line>
                                    </svg>
                                    <span core-show="topic-likes">{{$data->comments->count()}}</span>
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</article>