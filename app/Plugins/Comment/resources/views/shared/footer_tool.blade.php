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