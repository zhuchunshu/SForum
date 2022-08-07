<div class="card-footer">
    <div class="row">
        <div class="col">
            {{-- 点赞 --}}
            <a style="text-decoration:none;" core-click="like-topic" topic-id="{{ $data->id }}"
               class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom"
               title="{{__("topic.likes")}}">
                <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon" width="24" height="24"
                     viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                     stroke-linejoin="round">
                    <path stroke="none" d="M0 0h24v24H0z" fill="none"/>
                    <path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572"/>
                </svg>
                <span core-show="topic-likes">{{ $data->like }}</span>
            </a>
            {{-- markdown --}}
            <a data-bs-toggle="tooltip" data-bs-placement="top" title="" href="/{{ $data->id }}.md"
               data-bs-original-title="{{__("app.preview markdown")}}" class="hvr-icon-grow-rotate">
                    <span class="switch-icon-a text-muted">
                        <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-markdown"
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
            @if(auth()->check())
                {{--                收藏--}}
                <a style="text-decoration:none;" core-click="star-topic" topic-id="{{ $data->id }}"
                   class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom"
                   title="收藏">
                    <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-star"
                         width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                         stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z"></path>
                    </svg>
                </a>
                {{--                    举报--}}
                <a data-bs-toggle="modal" data-bs-target="#modal-report" style="text-decoration:none;"
                   core-click="report-topic" topic-id="{{ $data->id }}"
                   class="hvr-icon-pulse cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom"
                   title="举报">
                    <svg xmlns="http://www.w3.org/2000/svg" class="hvr-icon icon icon-tabler icon-tabler-flag-3"
                         width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                         stroke-linecap="round" stroke-linejoin="round">
                        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                        <path d="M5 14h14l-4.5 -4.5l4.5 -4.5h-14v16"></path>
                    </svg>
                </a>
            @endif
            {{--                引用--}}
            <a style="text-decoration:none;" core-click="copy" copy-content="[topic topic_id={{$data->id}}]" message="短代码复制成功!"
               class="hvr-icon-bounce cursor-pointer text-muted" data-bs-toggle="tooltip" data-bs-placement="bottom"
               title="引用">
                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-blockquote" width="24"
                     height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"
                     stroke-linecap="round" stroke-linejoin="round">
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

        {{--                    右边 footer--}}
        <div class="col-auto">
            {{--            修改记录--}}
            @if(count($data->topic_updated))
                <div class="avatar-list avatar-list-stacked">
                    @php $i = 1; @endphp
                    @foreach($data->topic_updated as $v)
                        @if($i<=5)
                            <span data-bs-toggle="modal" data-bs-target="#topic-updated"
                                  class="avatar avatar-sm avatar-rounded"
                                  style="--tblr-avatar-size:25px;background-image:url({{avatar_url($v->user_id)}})"></span>
                            @php
                                $i++;
                            @endphp
                        @endif

                    @endforeach
                    @if(count($data->topic_updated)>5)
                        <span class="avatar avatar-sm avatar-rounded" data-bs-toggle="modal"
                              data-bs-target="#topic-updated"
                              style="--tblr-avatar-size:25px;">+{{count($data->topic_updated)}}</span>
                    @endif
                </div>
            @endif

        </div>
    </div>
</div>
<div class="modal modal-blur fade" id="topic-updated" tabindex="-1" role="dialog" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" role="document">
        <div class="modal-content">
            <div class="modal-header">
                <h5 class="modal-title">帖子修订记录</h5>
                <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
            </div>
            <div class="modal-body">
                <ul class="list list-timeline">
                    @foreach($data->topic_updated as $value)
                        <li>
                            <div class="list-timeline-icon"><!-- Download SVG icon from http://tabler-icons.io/i/brand-twitter -->
                                <!-- SVG icon code -->
                                <a href="/users/{{$value->user->username}}.html" class="avatar avatar-rounded" style="background-image: url('{{avatar_url($value->user_id)}}')"></a>
                            </div>
                            <div class="list-timeline-content">
                                <div class="list-timeline-time">{{format_date($value->created_at)}}</div>
                                <p class="list-timeline-title">{{$value->user->username}}</p>
                                <p class="text-muted">修改时间:{{$value->created_at}}
                                    @if(get_options('topic_updated_author_ip','开启')==='开启' &&$value->user_ip)
                                        |
                                        <span class="text-red" topic-type="updated_ip" updated-id="{{$value->id}}">Loading<span class="animated-dots"></span></span>
                                    @endif
                                </p>
                            </div>
                        </li>
                    @endforeach

                </ul>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn me-auto" data-bs-dismiss="modal">Close</button>
                <button type="button" class="btn btn-primary" data-bs-dismiss="modal">好的</button>
            </div>
        </div>
    </div>
</div>

