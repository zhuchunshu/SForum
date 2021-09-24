<div class="row row-cards justify-content-center">
    <div class="col-md-12">
        <div class="border-0 card">
            <div class="card-body topic">
                @if($data->essence>0)
                    <div class="ribbon bg-green text-h3">
                        精华
                    </div>
                @endif
                <div class="row">
                    <div class="col-md-12" id="title">
                        <h1 data-bs-toggle="tooltip" data-bs-placement="top" title="帖子标题">

                            @if($data->topping>0)
                                <span class="text-red">
                                置顶
                            </span>
                            @endif

                            {{$data->title}}
                        </h1>
                    </div>
                    <div class="col-md-12">
                        @include('plugins.Core.topic.show.ol')
                    </div>
                    <hr class="hr-text" style="margin-top: 5px;margin-bottom: 5px">
                    <div class="col-md-12" id="author">
                        <div class="row">
                            <div class="col-auto">
                                <a class="avatar" href="/users/{{$data->user->username}}.html" style="background-image: url({{super_avatar($data->user)}})"></a>
                            </div>
                            <div class="col">
                                <div class="topic-author-name">
                                    <a href="/users/{{$data->user->username}}.html" class="text-reset">{{$data->user->username}}</a>
                                </div>
                                <div>发表于:{{format_date($data->created_at)}}</div>
                            </div>
                        </div>
                    </div>
                    <div class="col-md-12 vditor-reset" id="topic-content">
                        {!! ShortCodeR()->handle($data->content) !!}
                    </div>
                    @if($data->updated_user)
                        <div class="col-md-12">
                            <div class="hr-text hr-text-left">更新记录</div>
                            <div class="avatar-list avatar-list-stacked" style="margin-top: -15px">
                                <span data-bs-toggle="tooltip" data-bs-placement="top" title="{{$data->update_user->username}} 在:{{$data->updated_at}} 提交了一次更新" class="avatar" style="background-image: url({{super_avatar($data->update_user)}})" core-click="user-avatar" username="{{$data->update_user->username}}"></span>
                                @foreach($data->topic_updated as $value)
                                    @if($value->created_at!==$data->updated_at)
                                        <span data-bs-toggle="tooltip" data-bs-placement="top" title="{{$value->user->username}} 在:{{$value->created_at}} 提交了一次更新" class="avatar" style="background-image: url({{super_avatar($value->user)}})" core-click="user-avatar" username="{{$value->user->username}}"></span>
                                    @endif
                                @endforeach
{{--                                <span class="avatar">+8</span>--}}
                            </div>
                        </div>
                    @endif

                </div>
            </div>

            <div class="card-footer" style="background-color: #f4f6fa;">
{{--                点赞--}}
                <a style="text-decoration:none;" core-click="like-topic" topic-id="{{$data->id}}" class="text-muted cursor-pointer" data-bs-toggle="tooltip" data-bs-placement="bottom" title="点赞">
                    <svg xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none" /><path d="M19.5 13.572l-7.5 7.428l-7.5 -7.428m0 0a5 5 0 1 1 7.5 -6.566a5 5 0 1 1 7.5 6.572" /></svg>
                    <span core-show="topic-likes">{{$data->like}}</span>
                </a>
{{--                markdown --}}
                <a data-bs-toggle="tooltip" data-bs-placement="top" title="" href="/{{$data->id}}.md" data-bs-original-title="查看markdown文本">
                                    <span class="switch-icon-a text-muted">
                                        <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-markdown" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                            <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                            <rect x="3" y="5" width="18" height="14" rx="2"></rect>
                                            <path d="M7 15v-6l2 2l2 -2v6"></path>
                                            <path d="M14 13l2 2l2 -2m-2 2v-6"></path>
                                        </svg>
                                    </span>
                </a>
            </div>

        </div>
    </div>

</div>