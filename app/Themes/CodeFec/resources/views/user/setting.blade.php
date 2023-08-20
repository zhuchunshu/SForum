@extends("App::app")
@section('title','个人设置')
@section('content')
    <div>
        <div class="row row-cards justify-content-center">
            <div class="col-md-12">
                @if(count(Itf()->get('userSetting')))
                    <div class="card">
                        <div class="card-header">
                            <ul class="nav nav-pills card-header-pills">
                                @foreach(Itf()->get('userSetting') as $key=>$value)
                                    <li class="nav-item">
                                        <a class="nav-link @if(request()->input('m','userSetting_1')===$key) active @endif fw-bold"
                                           href="/user/setting?m={{$key}}">
                                            @if(\Hyperf\Collection\Arr::has($value,'icon'))
                                                {!! $value['icon'] !!}
                                            @endif
                                            {{$value['name']}}</a>
                                    </li>
                                @endforeach
                            </ul>
                        </div>
                        <div class="col-md-12">
                            @foreach(Itf()->get('userSetting') as $key=>$value)
                                @if(request()->input('m','userSetting_1')===$key)
                                    <div class="p-3">
                                        @include($value['view'],['data'=> $data])
                                    </div>
                                @endif
                            @endforeach
                        </div>
                    </div>
                @else
                    <div class="alert alert-warning">
                        <i class="fa fa-warning"></i>
                        <strong>无可设置项</strong>
                    </div>
                @endif
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix("plugins/Core/js/user.js")}}"></script>
@endsection