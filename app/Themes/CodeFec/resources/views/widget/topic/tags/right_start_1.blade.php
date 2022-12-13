<div class="col-md-12">
    <div class="card">

        <div class="card-status-top" style="{{Core_Ui()->Css()->bg_color($data->color)}}"></div>

        <div class="card-body">
            <h3 class="card-title">
                {{$data->name}}
            </h3>
            <p>
                {{core_default($data->description,get_options("description",__("app.no description")))}}
            </p>
            <b class="text-h3">创建者</b>:
            @if($data->user_id)
                <a href="/users/{{ $data->user->username }}.html">
                    <span class="avatar avatar-rounded" style="--tblr-avatar-size:20px;background-image: url({{super_avatar($data->user)}})"></span>
                    {{ $data->user->username }}</a>
            @else
                <span class="text-red">系统管理员</span>
            @endif
        </div>

        <div class="card-footer">
            @if(auth()->check())
                <a href="/topic/create" class="btn btn-dark">{{__("topic.create")}}</a>
                @if(Authority()->check('topic_tag_create') && $data->user_id == auth()->id())
                    <a href="/tags/{{$data->id}}/edit" class="btn btn-primary">{{__("app.edit")}}</a>
                @endif
            @else
                <a href="/login" class="btn btn-dark">{{__('app.login')}}</a>
                <a href="/register" class="btn btn-light">{{__("app.register")}}</a>
            @endif
        </div>

    </div>
</div>