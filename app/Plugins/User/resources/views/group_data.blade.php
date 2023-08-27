@extends("App::app")

@section('title', '「'.$data->name.'」'.__("user.user group info"))
@section('description', '为您展示本站「'.$data->name.'」'.__("user.user group info"))
@section('keywords', '为您展示本站「'.$data->name.'」'.__("user.user group info"))


@section('content')

    <div class="row justify-content-center row-cards">


        @if($userCount)
            <div class="col-md-12">
                <div style="margin-bottom: 10px">
                    @if($user->count())
                        <div class="card">
                            <div class="card-header">
                                <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                                    <li class="breadcrumb-item"><a href="/">首页</a></li>
                                    <li class="breadcrumb-item"><a href="/users">用户列表</a></li>
                                    <li class="breadcrumb-item active" aria-current="page"><a href="#">用户组</a></li>
                                    <li class="breadcrumb-item active" aria-current="page"><a href="#">{{$data->name}}</a></li>
                                </ol>

                            </div>
                            <div class="card-header">
                                <div class="row">
                                    <div class="col-auto"><span class="avatar bg-azure-lt">
                                {!! $data->icon?:'<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"></path>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>' !!}
                            </span>
                                        </div>
                                    <div class="col"><h3 class="card-title">{{$data->name}}</h3>
                                    <span class="text-red">{{__("app.permission value")}}:{{$data['permission-value']}}</span></div>
                                </div>
                                <div class="card-actions">
                                    @if($userCount)
                                        {{__("user.members in total",['total' => $userCount])}}
                                    @endif
                                </div>
                            </div>
                            <div class="card-body row row-cards">
                                @foreach($user as $value)
                                    <a href="/users/{{$value->id}}.html" class="col-md-5 m-1 card border-1 card-body"
                                       style="margin-bottom:5px;">
                                        <div class="row">
                                            <div class="col-auto">
                                                <span class="avatar avatar-md"
                                                      style="background-image: url({{super_avatar($value)}})"></span>
                                            </div>
                                            <div class="col">
                                                <span class="text-body h3 d-block text-truncate"><b>{{$value->username}}</b></span>
                                                {{__("user.registration date")}}:{{$value->created_at}}
                                            </div>
                                        </div>
                                    </a>
                                @endforeach
                            </div>
                        </div>
                    @else
                        <div class="card card-body">
                            <h3 class="card-title">{{__("app.No more results")}}</h3>
                        </div>
                    @endif
                </div>
                {!! make_page($user) !!}
            </div>
        @else
            <div class="col-12">
                <div class="card card-body">
                    <div class="card">
                        <div class="card-header">
                            <ol class="breadcrumb breadcrumb-arrows" aria-label="breadcrumbs">
                                <li class="breadcrumb-item"><a href="/">首页</a></li>
                                <li class="breadcrumb-item"><a href="/users">用户列表</a></li>
                                <li class="breadcrumb-item active" aria-current="page"><a href="#">用户组</a></li>
                                <li class="breadcrumb-item active" aria-current="page"><a href="#">{{$data->name}}</a></li>
                            </ol>

                        </div>
                        <div class="card-header">
                            <div class="row">
                                <div class="col-auto"><span class="avatar bg-azure-lt">
                                {!! $data->icon?:'<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0"></path>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>' !!}
                            </span>
                                </div>
                                <div class="col"><h3 class="card-title">{{$data->name}}</h3>
                                    <span class="text-red">{{__("app.permission value")}}:{{$data['permission-value']}}</span></div>
                            </div>
                            <div class="card-actions">
                                @if($userCount)
                                    ,{{__("user.members in total",['total' => $userCount])}}
                                @endif
                            </div>
                        </div>
                        <div class="card-body">
                            此用户组下暂无用户
                        </div>
                    </div>
                </div>
            </div>
        @endif

    </div>

@endsection
