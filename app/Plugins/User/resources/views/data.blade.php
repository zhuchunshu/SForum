@extends("Core::app")

@section('title', '「'.$data->username.'」会员的信息')
@section('description', '为您展示本站「'.$data->username.'」用户的信息')
@section('keywords', '为您展示本站「'.$data->username.'」用户的信息')


@section('content')

    <div class="row justify-content-center row-cards">
        <div class="col-md-10">
            <div class="card card-stacked">
                <div class="card-body">
                    <div class="row g-2 align-items-center">
                        <div class="col-auto">
                                {!! avatar($data->id,"avatar-lg") !!}
                        </div>
                        <div class="col">
                            <h4 class="card-title m-0">
                                <a href="/users/{{$data->username}}.html">{{$data->username}}</a>

                                <a href="/users/group/{{$data->Class->id}}.html">{!! Core_Ui()->Html()->UserGroup($data->Class) !!}</a>

{{--                                <span class="badge bg-pink">MAX</span>--}}
                            </h4>
                            <div class="text-muted">
{{--                                <svg--}}
{{--                                        xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"--}}
{{--                                        viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"--}}
{{--                                        stroke-linecap="round" stroke-linejoin="round">--}}
{{--                                    <path stroke="none" d="M0 0h24v24H0z" fill="none" />--}}
{{--                                    <circle cx="12" cy="11" r="3" />--}}
{{--                                    <path--}}
{{--                                            d="M17.657 16.657l-4.243 4.243a2 2 0 0 1 -2.827 0l-4.244 -4.243a8 8 0 1 1 11.314 0z" />--}}
{{--                                </svg>山东                                    <svg--}}
{{--                                        xmlns="http://www.w3.org/2000/svg" class="icon" width="24" height="24"--}}
{{--                                        viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none"--}}
{{--                                        stroke-linecap="round" stroke-linejoin="round">--}}
{{--                                    <path stroke="none" d="M0 0h24v24H0z" fill="none" />--}}
{{--                                    <line x1="3" y1="21" x2="21" y2="21" />--}}
{{--                                    <line x1="9" y1="8" x2="10" y2="8" />--}}
{{--                                    <line x1="9" y1="12" x2="10" y2="12" />--}}
{{--                                    <line x1="9" y1="16" x2="10" y2="16" />--}}
{{--                                    <line x1="14" y1="8" x2="15" y2="8" />--}}
{{--                                    <line x1="14" y1="12" x2="15" y2="12" />--}}
{{--                                    <line x1="14" y1="16" x2="15" y2="16" />--}}
{{--                                    <path d="M5 21v-16a2 2 0 0 1 2 -2h10a2 2 0 0 1 2 2v16" />--}}
{{--                                </svg>CodeFec                                            #CTO                                                                        发帖总量:48--}}
{{--                                共发布了65条评论,--}}
                                @if($data->Options->email)
                                    邮箱: <a href="mailto:{{$data->Options->email}}">{{$data->Options->email}}</a>,
                                @endif
                                注册于:{{format_date($data->created_at)}}@if($data->updated_at),最后更新:{{format_date($data->updated_at)}},{{$data->fans}}个粉丝@endif
                            </div>
                            <div>
                                <a href="/users/fans/{{$data->username}}.html" class="btn btn-primary btn-sm btn-square">TA的粉丝</a>
                                <a href="/users/topic/{{$data->username}}.html" class="btn btn-primary btn-sm btn-square">TA的帖子</a>
                            </div>

                        </div>
                        <div class="col-auto">
                            <a class="btn btn-danger cursor-pointer" user-click="user_follow" user-id="{{ $data->id }}">
                                <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-user-plus" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                    <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                                    <circle cx="9" cy="7" r="4"></circle>
                                    <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
                                    <path d="M16 11h6m-3 -3v6"></path>
                                </svg>
                                <span>关注</span>
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="col-md-10">
            <div class="card card-stacked card-body">
                <h3 class="card-title">签名</h3>
                <div class="markdown">
                    {!! markdown()->text(core_default($data->Options->qianming,"暂无无签名")) !!}
                </div>
            </div>
        </div>

        @if($data->Options->qq || $data->Options->wx || $data->Options->website || $data->Options->email)
        <div class="col-md-10">
                <div class="card card-stacked card-body">
                    <h3 class="card-title">其他信息</h3>
                    @if($data->Options->qq)
                        <div class="mb-2">
                            QQ: {{$data->Options->qq}}
                        </div>
                    @endif

                    @if($data->Options->wx)
                        <div class="mb-2">
                            微信: {{$data->Options->wx}}
                        </div>
                    @endif

                    @if($data->Options->email)
                        <div class="mb-2">
                            邮箱: {{$data->Options->email}}
                        </div>
                    @endif

                    @if($data->Options->website)
                        <div class="mb-2">
                            个人网站: <a href="{{$data->Options->website}}">{{$data->Options->website}}</a>
                        </div>
                    @endif
                </div>
        </div>
        @endif
    </div>


@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection