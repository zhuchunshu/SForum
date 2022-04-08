@extends("App::app")

@section('title', '「'.$data->name.'」用户组的信息')
@section('description', '为您展示本站「'.$data->name.'」用户组的信息')
@section('keywords', '为您展示本站「'.$data->name.'」用户组的信息')


@section('content')

    <div class="row justify-content-center row-cards">

        <div class="col-md-12">
            <div class="border-0 card">
                <div class="card-status-top" style="{{ Core_Ui()->Css()->bg_color($data->color) }}"></div>
                <div class="card-body">
                    <div class="row">
                        <div class="col-auto">
                            <span class="avatar" style="background-color: {{$data->color}}"></span>
                        </div>
                        <div class="col">
                            <div class="row"><div class="col-auto"><h2>{{$data->name}}</h2></div><div class="col text-red">权限值:{{$data['permission-value']}}</div></div>
                            创建日期:{{$data->created_at}} @if($data->updated_at), 最后更新时间:{{$data->updated_at}}@endif
                            @if($userCount),该用户组下一共有 {{$userCount}} 位用户@endif
                        </div>
                    </div>
                </div>
            </div>
        </div>

        @if($userCount)
            <div class="col-md-12">
                <div style="margin-bottom: 10px">
                    @if($user->count())
                        <div class="border-0 card">
                            <div class="card-status-top" style="{{ Core_Ui()->Css()->bg_color($data->color) }}"></div>
                            <div class="card-body">
                                <h3 class="card-title">该用户组下的会员列表</h3>
                                <div class="row">
                                    @foreach($user as $value)
                                    <div class="col-md-6" style="margin-bottom:5px">
                                        <div class="row">
                                            <div class="col-auto">
                                                {!! avatar($value->id) !!}
                                            </div>
                                            <div class="col">
                                                <a href="/users/{{$value->username}}.html" class="text-body d-block text-truncate"><b>{{$value->username}}</b></a>
                                                注册日期:{{$value->created_at}}
                                            </div>
                                        </div>
                                    </div>
                                    @endforeach
                                </div>
                            </div>
                        </div>
                    @else
                        <div class="card card-body">
                            <h3 class="card-title">无更多结果</h3>
                        </div>
                    @endif
                </div>
                {!! make_page($user) !!}
            </div>
        @endif

    </div>


@endsection
