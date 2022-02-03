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
                @if(auth()->check())
                    {{--                收藏--}}
                    <a style="text-decoration:none;" core-click="star-topic" topic-id="{{ $data->id }}"
                       class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="收藏">
                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-star" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                        </svg>
                    </a>
{{--                    举报--}}
                    <a data-bs-toggle="modal" data-bs-target="#modal-report" style="text-decoration:none;" core-click="report-topic" topic-id="{{ $data->id }}"
                       class="hvr-icon-pulse cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom" title="举报">
                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-flag-3" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                            <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
                        </svg>
                    </a>

                @endif
            </div>

        </div>

    </div>

{{--    上下页--}}
    @include('Core::topic.show.include.lfpage')
{{--    显示评论--}}
    @include('Comment::Widget.show-topic')
{{--    评论--}}
    @include('Comment::Widget.topic')

    @if(auth()->check())
{{--        举报模态--}}
        <div class="modal fade" id="modal-report" tabindex="-1" role="dialog" aria-hidden="true">
            <div class="modal-dialog modal-dialog-centered" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title" id="modal-report-title"></h5>
                        <button type="button" modal-click="close" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div class="row mb-3 align-items-end">
                            <div class="col-auto">
                                <label class="form-label">举报原因</label>
                                <select class="form-select" id="modal-report-select">
                                    <option value="水贴">水贴</option>
                                    <option value="广告">广告</option>
                                    <option value="引战">引战</option>
                                    <option value="违法">违法</option>
                                    <option value="翻墙">翻墙</option>
                                    <option value="政治">政治</option>
                                    <option value="其他">其他</option>
                                </select>
                            </div>
                            <div class="col">
                                <label class="form-label">标题</label>
                                <input type="text" id="modal-report-input-title" class="form-control" />
                                <input type="hidden" value="" id="modal-report-input-type" />
                                <input type="hidden" value="" id="modal-report-input-type-id" />
                                <input type="hidden" value="" id="modal-report-input-url" />
                            </div>
                        </div>
                        <div>
                            <label class="form-label">详细说明</label>
                            <textarea class="form-control" id="modal-report-input-content"></textarea>
                            <small><b style="color: red">支持markdown</b></small>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn me-auto" data-bs-dismiss="modal" modal-click="close">Close</button>
                        <button type="button" class="btn btn-primary" data-bs-dismiss="modal" id="modal-report-submit">提交</button>
                    </div>
                </div>
            </div>
        </div>
    @endif
</div>

