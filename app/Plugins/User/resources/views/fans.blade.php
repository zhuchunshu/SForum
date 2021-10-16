@extends("Core::app")

@section('title',"「".$user->username."」的粉丝列表")
@section('description','为你展示「'.$user->username.'」的粉丝列表')

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
                            {{$user->username}} 的粉丝列表
                        </h2>

                    </div>


                </div>
            </div>
        </div>
    </div>
@endsection

@section('content')

    <div class="row row-cards">
        @if ($page->count())
            @foreach ($page as $value)
                <div class="col-md-6 col-lg-3">
                    <div class="border-0 card">
                        <div class="card-body p-4 text-center">
                            {!! avatar($value->fans->id,"avatar-xl mb-3 avatar-rounded") !!}
                            <h3 class="m-0 mb-1">
                                <a href="/users/{{$value->fans->username}}.html">{{$value->fans->username}}</a>
                            </h3>
                            <div class="text-muted">本站第{{$value->fans->id}}位会员</div>
                            <div class="text-muted">注册时间:{{$value->fans->created_at}}</div>
                            <div class="text-muted">关注时间:{{$value->created_at}}</div>
                            <div class="mt-3">
                                <a href="/users/group/{{$value->fans->Class->id}}.html">
                                    {!! Core_Ui()->Html()->UserGroup($value->fans->Class) !!}
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
            @endforeach
        @else
            <div class="col-md-4">
                <div class="border-0 card">
                    <div class="card-status-top bg-danger"></div>
                    <div class="card-body">
                        <div class="row">
                            <div class="col-auto">
                                <span class="avatar"></span>
                            </div>
                            <div class="col">
                                <h3 class="card-title text-h2">暂无内容</h3>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        @endif
        {!! make_page($page) !!}
    </div>

@endsection

@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection
@section('scripts')
    <script src="{{mix('plugins/Topic/js/core.js')}}"></script>
@endsection
