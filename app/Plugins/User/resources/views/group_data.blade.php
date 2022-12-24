@extends("App::app")

@section('title', '「'.$data->name.'」'.__("user.user group info"))
@section('description', '为您展示本站「'.$data->name.'」'.__("user.user group info"))
@section('keywords', '为您展示本站「'.$data->name.'」'.__("user.user group info"))


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
                            <div class="row"><div class="col-auto"><h2>{{$data->name}}</h2></div><div class="col text-red">
                                {{__("app.permission value")}}:{{$data['permission-value']}}</div></div>
                            {{__("created at",['time' => $data->created_at])}} @if($data->updated_at), {{__("app.last update time",['time' => $data->updated_at])}}@endif
                            @if($userCount),{{__("user.members in total",['total' => $userCount])}}@endif
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
                                <h3 class="card-title">{{__("user.List of members under this user group")}}</h3>
                                <div class="row">
                                    @foreach($user as $value)
                                    <div class="col-md-6" style="margin-bottom:5px">
                                        <div class="row">
                                            <div class="col-auto">
                                                <span class="avatar" style="background-image: url({{super_avatar($value)}})"></span>
                                            </div>
                                            <div class="col">
                                                <a href="/users/{{$value->username}}.html" class="text-body d-block text-truncate"><b>{{$value->username}}</b></a>
                                                {{__("user.registration date")}}:{{$value->created_at}}
                                            </div>
                                        </div>
                                    </div>
                                    @endforeach
                                </div>
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
        @endif

    </div>


@endsection
