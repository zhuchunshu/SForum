@extends("App::app")

@section('title', '「'.$data->name.'」板块的相关信息,以及此板块下的所有帖子')
@section('description', '为您展示「'.$data->name.'」板块的相关信息,以及此板块下的所有帖子')
@section('keywords', '为您展示「'.$data->name.'」板块的相关信息,以及此板块下的所有帖子')

@section('content')
    <div class="row row-cards justify-content-center">
        <div class="col-md-12">
            <div class="row row-cards justify-content-center">
                <div class="col-lg-9">
                    @include('Topic::Tags.data.index')
                </div>
                <div class="col-lg-3">
                    <div class="row row-cards @if(get_options('theme_right_tool_sticky')!=='true'){{"rd"}}@endif">
                        <div class="col-md-12 @if(get_options('theme_right_tool_sticky')!=='true'){{"sticky"}}@endif" style="top: 105px">
                            @include('Topic::Tags.data.right')
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    @if($data->moderator->count())

        <div class="modal modal-blur fade" id="modal-moderator-list" tabindex="-1" role="dialog" aria-hidden="true">
            <div class="modal-dialog modal-lg modal-dialog-centered" role="document">
                <div class="modal-content">
                    <div class="modal-header">
                        <h5 class="modal-title">版主列表</h5>
                        <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                    </div>
                    <div class="modal-body">
                        <div class="list-group list-group-flush">
                            @foreach($data->moderator as $moderator)
                                <div class="list-group-item">
                                    <div class="row">
                                        <div class="col-auto">
                                            <a href="/users/{{$moderator->user->id}}.html" class="avatar" style="background-image: url({{avatar($moderator->user)}})"></a>
                                        </div>
                                        <div class="col text-truncate">
                                            <a href="/users/{{$moderator->user->id}}.html" class="text-body d-block">{{$moderator->user->username}}</a>
                                            <div class="text-muted text-truncate mt-n1">@if($moderator->user->Options->qianming && $moderator->user->Options->qianming!=='no bio') {{$moderator->user->Options->qianming}}@else{{"没有签名"}}@endif</div>
                                        </div>
                                    </div>
                                </div>
                            @endforeach
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button type="button" class="btn me-auto" data-bs-dismiss="modal">Close</button>
                        <button type="button" class="btn btn-primary" data-bs-dismiss="modal">好的</button>
                    </div>
                </div>
            </div>
        </div>
    @endif
@endsection


@section('headers')
    <link rel="stylesheet" href="{{ mix('plugins/Topic/css/app.css') }}">
@endsection