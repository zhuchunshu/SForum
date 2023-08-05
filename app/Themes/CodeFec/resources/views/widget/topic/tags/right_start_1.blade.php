<div class="col-md-12">
    <a href="/tags/{{$data->id}}.html" class="card card-link text-primary-fg" style="background-color: {{$data->color}}!important;">
        <div class="card-stamp">
            <div class="card-stamp-icon bg-yellow">
                <!-- Download SVG icon from http://tabler-icons.io/i/star -->
                {!! $data->icon !!}
            </div>
        </div>
        <div class="card-body">
            <h3 class="card-title">{{$data->name}}</h3>
            <p>{{ \Hyperf\Stringable\Str::limit(core_default($data->description, __("app.no description")), 32) }}</p>
        </div>
    </a>
</div>
<div class="col-md-12">
    <div class="card">

        <div class="card-status-top" style="{{Core_Ui()->Css()->bg_color($data->color)}}"></div>

        <div class="card-body">
            <b class="text-h3">标签创建者</b>:
            @if($data->user_id)
                <a href="/users/{{ $data->user->id }}.html">
                    <span class="avatar avatar-rounded" style="--tblr-avatar-size:20px;background-image: url({{super_avatar($data->user)}})"></span>
                    {{ $data->user->username }}</a>
            @else
                <span class="text-red">系统管理员</span>
            @endif
            @if($data->moderator->count())
            <div class="mt-3">
                <b class="text-h3">版主：</b>
                <div data-bs-toggle="modal" data-bs-target="#modal-moderator-list" class="avatar-list avatar-list-stacked mt-2">
                    @foreach($data->moderator as $moderator)
                        <span class="avatar avatar-sm rounded-circle" style="background-image: url({{avatar($moderator->user)}})"></span>
                    @endforeach
                </div>
            </div>
            @endif
        </div>

        <div class="card-footer">
            @if(auth()->check())
                <a href="/topic/create?basis[tag]={{$data->id}}" class="btn btn-dark">{{__("topic.create")}}</a>
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
