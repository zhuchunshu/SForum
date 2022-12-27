@extends("App::app")

@section('title', __("topic.create"))

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
                            {{__("topic.create")}}
                        </h2>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection
@section('content')

    <div class="row row-cards">

        <div class="col-12" id="topic-create">
            <form action="/topic/create" method="POST">
                <div class="row row-cards">
                    <div class="col-lg-9">

                        <div class="card">
                            <div class="card-header">
                                <h3 class="card-title">帖子信息</h3>
                            </div>
                            <div class="card-body">
                                @foreach(Itf()->get('topic-create-data') as $k=>$v)
                                    @if(call_user_func($v['enable'])===true && isset($v['view']))
                                        @include($v['view'])
                                    @endif
                                @endforeach
                            </div>
                        </div>

                    </div>

                    <div class="col-lg-3">
                        <div class="card">
                            <div class="card-header">
                                <h3 class="card-title">附加信息</h3>
                            </div>
                            <div class="card-body">
                                <div class="row row-cards">
                                    @foreach(Itf()->get('topic-create-options') as $k=>$v)
                                        @if(call_user_func($v['enable'])===true)
                                            @include($v['view'])
                                        @endif
                                    @endforeach
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-12">
                        <div class="row row-cards">
                            <div class="col-lg-9">
                                <div class="card card-body">
                                    <label for="" class="form-label">验证码</label>
                                    <div class="input-group">
                                        <input type="text" name="captcha" class="form-control" placeholder="captcha"
                                               autocomplete="off" required>
                                        <span class="input-group-link">
                                 <img class="captcha" src="{{captcha()->inline()}}" alt=""
                                      onclick="this.src='/captcha?id='+Math.random()">
                            </span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="col-12">
                        <x-csrf/>
                        <button class="btn btn-primary" type="submit">提交</button>
                    </div>
                </div>
            </form>
        </div>
    </div>

@endsection

@section('scripts')
    @foreach(Itf()->get('topic-create-data') as $k=>$v)
        @if(call_user_func($v['enable'])===true && isset($v['scripts']))
            @foreach($v['scripts'] as $script)
                <script src="{{$script}}" defer></script>
            @endforeach
        @endif
    @endforeach
    @foreach(Itf()->get('topic-create-options') as $k=>$v)
        @if(call_user_func($v['enable'])===true && isset($v['scripts']) && is_array($v['scripts']))
            @foreach($v['scripts'] as $script)
                <script src="{{$script}}" defer></script>
            @endforeach
        @endif
    @endforeach
@endsection