@extends("App::app")

@section('title', '「'.$user->username.'」的收藏')

@section('header')
    <div class="page-wrapper">
        <div class="container-xl">
            <!-- Page title -->
            <div class="page-header d-print-none">
                <div class="row align-items-center">
                    <div class="col">
                        <!-- Page pre-title -->
                        <div class="page-pretitle">
                            Overview
                        </div>
                        <h2 class="page-title">
                            {{$user->username}}的收藏
                        </h2>
                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')
    <div class="row row-cards justify-content-center">
        @if($page->count())
            @foreach($page as $value)
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
                                            <span class="home-summary">{{ \Hyperf\Utils\Str::limit(core_default(deOptions(get_topic($value->type_id)->options)["summary"],"未捕获到本文摘要"),300) }}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="card-footer">
                                <div class="row">
                                    <div class="col">
                                        <a href="/users/{{get_topic($value->type_id)->user->username}}.html">
                                            <span class="avatar"
                                                  style="background-image: url({{super_avatar(get_topic($value->type_id)->user)}})">

                                         </span>
                                        </a>
                                    </div>
                                    <div class="col-auto">
                                        <a href="/{{$value->type_id}}.html" class="btn btn-primary">查看</a>
                                        @if($quanxian)
                                            <button user-click="remove_collections" collections-id="{{$value->id}}"
                                                    class="btn btn-danger">取消收藏
                                            </button>
                                        @endif
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
                                                                <a href="/users/{{get_topic_comment($value->type_id)->user->username}}.html"><span
                                                                            class="avatar"
                                                                            style="background-image: url({{super_avatar(get_topic_comment($value->type_id)->user)}})">

                                                        </span></a>
                                                            </div>
                                                            {{--                                            作者信息--}}
                                                            <div class="col text-truncate">
                                                                <a style="white-space:nowrap;"
                                                                   href="/users/{{get_topic_comment($value->type_id)->user->username}}.html"
                                                                   class="text-body text-truncate">{{get_topic_comment($value->type_id)->user->username}}</a>
                                                                <br/>
                                                                <small data-bs-toggle="tooltip" data-bs-placement="top"
                                                                       data-bs-original-title="{{$value->created_at}}"
                                                                       class="text-muted text-truncate mt-n1">发表于:{{format_date(get_topic_comment($value->type_id)->created_at)}}</small>
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
                                                            评论内容
                                                        </div>
                                                    </div>
                                                    <div core-show="comment" comment-id="{{$value->type_id}}"
                                                         class="col-md-12 markdown vditor-reset">
                                                        @if(get_topic_comment($value->type_id)->parent_id)
                                                            <div class="quote">
                                                                <blockquote>
                                                                    <a style="font-size:13px;"
                                                                       href="{{get_topic_comment($value->type_id)->parent_url}}" target="_blank">
                                                                        <span style="color:#999999">{{get_topic_comment($value->type_id)->parent->user->username}} 发表于 {{format_date(get_topic_comment($value->type_id)->created_at)}}</span>
                                                                    </a>
                                                                    <br>
                                                                    {!! \Hyperf\Utils\Str::limit(remove_bbCode(strip_tags(get_topic_comment($value->type_id)->parent->content)),60) !!}
                                                                </blockquote>
                                                            </div>
                                                        @endif
                                                        {!! ShortCodeR()->handle(get_topic_comment($value->type_id)->content) !!}
                                                    </div>


                                                </div>
                                            </div>

                                        </div>
                                    </div>
                                    <div class="card-footer">
                                        <div class="row">
                                            <div class="col">
                                                <a href="/users/{{get_topic_comment($value->type_id)->user->username}}.html">
                                            <span class="avatar"
                                                  style="background-image: url({{super_avatar(get_topic_comment($value->type_id)->user)}})">

                                         </span>
                                                </a>
                                            </div>
                                            <div class="col-auto">
                                                <a href="/{{get_topic_comment($value->type_id)->topic_id}}.html/{{$value->type_id}}?page={{get_topic_comment_page($value->type_id)}}" class="btn btn-primary">查看</a>
                                                @if($quanxian)
                                                    <button user-click="remove_collections" collections-id="{{$value->id}}"
                                                            class="btn btn-danger">取消收藏
                                                    </button>
                                                @endif
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    @endif

                </div>
            @endforeach
        @else
            <div class="col-md-12">
                <div class="border-0 card">
                    <div class="card-body">
                        <h3 class="card-title">暂无收藏内容</h3>
                    </div>
                </div>
            </div>
        @endif
        {!! make_page($page) !!}
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection