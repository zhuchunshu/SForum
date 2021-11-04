<div class="row row-cards justify-content-center">
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body topic">
                @if ($data->essence > 0)
                    <div class="ribbon bg-green text-h3">
                        精华
                    </div>
                @endif
                <div class="row">
                    <div class="col-md-12" id="title">
                        <h1 data-bs-toggle="tooltip" data-bs-placement="left" title="帖子标题">

                            @if ($data->topping > 0)
                                <span class="text-red">
                                    置顶
                                </span>
                            @endif

                            {{ $data->title }}
                        </h1>
                    </div>
                    <div class="col-md-12">
                        @include('Core::topic.show.ol')
                    </div>
                    <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">
                    <div class="col-md-12" id="author">
                        <div class="row">
                            <div class="col-auto">
                                <a class="avatar" href="/users/{{ $data->user->username }}.html"
                                    style="background-image: url({{ super_avatar($data->user) }})"></a>
                            </div>
                            <div class="col">
                                <div class="topic-author-name">
                                    <a href="/users/{{ $data->user->username }}.html"
                                        class="text-reset">{{ $data->user->username }}</a>
                                </div>
                                <div>发表于:{{ format_date($data->created_at) }}</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-12 vditor-reset" id="topic-content">
                        {!! ShortCodeR()->handle($data->content) !!}
                    </div>

                </div>
            </div>

            <div class="card-footer" style="background-color: #f4f6fa;">
                {{-- 点赞 --}}
                <a style="text-decoration:none;" core-click="like-topic" topic-id="{{ $data->id }}"
                    class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="点赞">
                    <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon" width="24" height="24"
                        viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                        stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none" />
                        <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" />
                    </svg>
                    <span core-show="topic-likes">{{ $data->like }}</span>
                </a>
                {{-- markdown --}}
                <a data-bs-toggle="tooltip" data-bs-placement="top" title="" href="/{{ $data->id }}.md"
                    data-bs-original-title="查看markdown文本" class="hvr-icon-grow-rotate">
                    <span class="switch-icon-a text-muted">
                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-markdown" width="24"
                            height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                            stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                            <path d="M7 15v-6l2 2l2 -2v6"></path>
                            <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                        </svg>
                    </span>
                </a>
{{--                收藏--}}
                <a style="text-decoration:none;" core-click="star-topic" topic-id="{{ $data->id }}"
                   class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="收藏">
                    <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-star" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                    </svg>
                </a>
            </div>

        </div>

    </div>

{{--    上下页--}}
    @include('Core::topic.show.include.lfpage')
{{--    显示评论--}}
    @include('Comment::Widget.show-topic')
{{--    评论--}}
    @include('Comment::Widget.topic')



</div>

