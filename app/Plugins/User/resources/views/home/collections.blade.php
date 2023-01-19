<link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
<div class="row row-cards">

    @php($collections = \App\Plugins\User\src\Models\UsersCollection::query()->where(['user_id' => $user->id])->paginate(15))
    <div class="col-md-12">
        <div class="border-0 card card-body">
            <h3 class="card-title">{{$user->username}} 的收藏</h3>
            @if($collections->count())
                <div class="row row-cards">
                    @foreach($collections as $value)
                        <div class="col-md-12">
                            @if($value->type==="topic")
                                <div class="card">
                                    <div class="card-header">
                                        <h3 class="card-title">
                                            <a href="/{{$value->type_id}}.html">{{get_topic($value->type_id)->title}}</a>
                                        </h3>
                                    </div>
                                    <div class="card-body">
                                        <div class="row">
                                            <div class="col-md-12">
                                                <div class="card-text">
                                                    <span class="home-summary markdown">{!! content_brief(get_topic($value->type_id)->post->content,get_options("topic_brief_len",250)) !!}</span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                    <div class="card-footer">
                                        <div class="row">
                                            <div class="col">
                                                <a href="/users/{{get_topic($value->type_id)->user->id}}.html">
                                            <span class="avatar"
                                                  style="background-image: url({{super_avatar(get_topic($value->type_id)->user)}})">

                                         </span>
                                                </a>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            @endif

                            @if($value->type==="comment")
                                <div class="row row-cards justify-content-center">
                                    <div class="col-md-12">
                                        <div class="card">
                                            <div id="comment-{{get_topic_comment($value->type_id)->id}}" name="comment-{{get_topic_comment($value->type_id)->id}}" class="col-md-12">
                                                <div class="card @if(get_topic_comment($value->type_id)->optimal) comment-optimal @else border-0 @endif">
                                                    <div class="card-body">
                                                        <div class="row">
                                                            {{--                                    作者信息--}}
                                                            <div class="col-md-12">
                                                                <div class="row">
                                                                    {{--                                            头像--}}
                                                                    <div class="col-auto"
                                                                         id="comment-user-avatar-{{$value->type_id}}"
                                                                         comment-show="user-data" user-id="{{get_topic_comment($value->type_id)->user_id}}">
                                                                        <a href="/users/{{get_topic_comment($value->type_id)->user->id}}.html"><span
                                                                                    class="avatar"
                                                                                    style="background-image: url({{super_avatar(get_topic_comment($value->type_id)->user)}})">

                                                        </span></a>
                                                                    </div>
                                                                    {{--                                            作者信息--}}
                                                                    <div class="col text-truncate">
                                                                        <a style="white-space:nowrap;"
                                                                           href="/users/{{get_topic_comment($value->type_id)->user->id}}.html"
                                                                           class="text-body text-truncate">{{get_topic_comment($value->type_id)->user->username}}</a>
                                                                        <br/>
                                                                        <small data-bs-toggle="tooltip" data-bs-placement="top"
                                                                               data-bs-original-title="{{$value->created_at}}"
                                                                               class="text-muted text-truncate mt-n1">{{__("app.Published on")}}:{{format_date(get_topic_comment($value->type_id)->created_at)}}</small>
                                                                    </div>
                                                                    {{--                                            楼层信息--}}
                                                                    <div class="col-auto">
                                                                        <a class="badge badge-pill bg-teal"
                                                                           href="/{{get_topic_comment($value->type_id)->topic_id}}.html">访问所在帖子</a>
                                                                    </div>

                                                                </div>
                                                            </div>
                                                            {{--                                    评论内容--}}
                                                            <div class="col-md-12">
                                                                <div class="hr-text" style="margin-bottom:8px;margin-top:15px">
                                                                    {{__("topic.comment.comment content")}}
                                                                </div>
                                                            </div>
                                                            <div core-show="comment" comment-id="{{$value->type_id}}"
                                                                 class="col-md-12 markdown">
                                                                @if(get_topic_comment($value->type_id)->parent_id)
                                                                    <div class="quote">
                                                                        <blockquote>
                                                                            <a style="font-size:13px;"
                                                                               href="{{get_topic_comment($value->type_id)->parent_url}}" target="_blank">
                                                                                <span style="color:#999999">{{get_topic_comment($value->type_id)->parent->user->username}} {{__("app.Published on")}} {{format_date(get_topic_comment($value->type_id)->created_at)}}</span>
                                                                            </a>
                                                                            <br>
                                                                            {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags(get_topic_comment($value->type_id)->parent->post->content)),60) !!}
                                                                        </blockquote>
                                                                    </div>
                                                                @endif
                                                                    {!!CommentContentParse()->parse(get_topic_comment($value->type_id)->post->content,['comment' => get_topic_comment($value->type_id),'topic' =>get_topic_comment($value->type_id)->topic,'remove_shortCode' => ['topic-comment']]) !!}
                                                            </div>


                                                        </div>
                                                    </div>

                                                </div>
                                            </div>
                                            <div class="card-footer">
                                                <div class="row">
                                                    <div class="col">
                                                        <a href="/users/{{get_topic_comment($value->type_id)->user->id}}.html">
                                            <span class="avatar"
                                                  style="background-image: url({{super_avatar(get_topic_comment($value->type_id)->user)}})">

                                         </span>
                                                        </a>
                                                    </div>
                                                    <div class="col-auto">
                                                        <a href="/{{get_topic_comment($value->type_id)->topic_id}}.html/{{$value->type_id}}?page={{get_topic_comment_page($value->type_id)}}" class="btn btn-primary">查看</a>

                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            @endif

                        </div>
                    @endforeach
                </div>
                <div class="mt-2">
                    {!! make_page($collections) !!}
                </div>
            @else
                <div class="empty">
                    <div class="empty-header">404</div>
                    <p class="empty-title">暂无更多结果</p>
                </div>
            @endif
        </div>
    </div>
</div>