<div class="row row-cards justify-content-center">
    <div class="col-md-12" id="topic">
        <div class="card">
            <div class="card-header">
                <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                    <li class="breadcrumb-item"><a href="/">首页</a></li>
                    <li class="breadcrumb-item"><a href="/tags/{{$data->tag->id}}.html">
                            {!! $data->tag->icon !!}
                            {{$data->tag->name}}
                        </a></li>
                    <li class="breadcrumb-item active" aria-current="page"><a href="">{{\Hyperf\Utils\Str::limit($data->title,20)}}</a></li>
                </ol>
            </div>
            <div class="card-body topic">
                @if ($data->essence > 0)
                    <div class="ribbon bg-green text-h3">
                        {{__('app.essence')}}
                    </div>
                @endif
                <div class="row" style="margin-top: -10px">
                    {{--                    标题--}}
                    <div class="col-md-12" id="title">
                        <h1 data-bs-toggle="tooltip" data-bs-placement="left" title="{{__('topic.title')}}">
                            @if ($data->topping > 0)
                                <span class="text-red">
                                    {{__('app.top')}}
                                </span>
                            @endif
                            {{ $data->title }}
                        </h1>
                    </div>

                    {{--                    面包屑--}}
                    <div class="col-md-12">
                        @include('App::topic.show.ol')
                    </div>
                    <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">

                    {{--                    作者信息--}}
                    @include('App::topic.show.include.author')

                    {{--文章信息--}}
                    <article class="col-md-12 article markdown" id="topic-content">
                        {!! ContentParse()->parse($data->post->content,$parseData) !!}
                    </article>
                    @if($data->user->Options->qianming!=='no bio')
                        <div class="hr-text hr-text-left mb-3">signature</div>
                        <span class="text-muted mb-0">
                            {{$data->user->Options->qianming}}
                        </span>
                    @endif

                </div>
            </div>

            {{--            页脚--}}
            @include('App::topic.show.include.footer')

        </div>

    </div>

    {{--    上下页--}}
    @include('App::topic.show.include.lfpage')
    @if(!isset($data->post->options->disable_comment))
        {{--    显示评论--}}
        @include('Comment::Widget.show-topic')
        {{--    评论--}}
        @include('Comment::Widget.topic')
    @else
        @if(@$data->post->options->disable_comment)
            <div class="col-md-12">
                <div class="border-0 card">
                    <div class="empty">
                        <div class="empty-icon"><!-- Download SVG icon from http://tabler-icons.io/i/mood-sad -->
                            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-ban" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
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
                                    <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-login" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
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
            {{--    评论--}}
            @include('Comment::Widget.topic')
        @endif
    @endif

    @if(auth()->check())
        {{--        举报模态--}}
        @include('App::topic.show.include.report')
    @endif
</div>

